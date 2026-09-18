package tripo3d

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// retrySafety describes what a failure tells us about whether the server
// acted on the request. Task-creation endpoints are billed per submission,
// so replaying a request that may already have been processed can charge the
// caller twice.
type retrySafety int

const (
	// safetyFatal is a non-transient failure: never retry.
	safetyFatal retrySafety = iota
	// safetyUnknown means the request may or may not have been processed.
	// Only idempotent methods may be retried.
	safetyUnknown
	// safetyClean means the server provably never acted on the request, so
	// a retry is safe regardless of method.
	safetyClean
)

// Statuses where the server answered and told us it declined to do the work.
var cleanRetryStatuses = map[int]bool{429: true, 503: true}

// Statuses where the server answered but whether it processed the request is
// unknowable — a 504 in particular is often emitted by a proxy after the
// origin already accepted the work.
var unknownRetryStatuses = map[int]bool{408: true, 425: true, 500: true, 502: true, 504: true}

const indeterminateHint = "the server may already have accepted this request, so it was not retried automatically; " +
	"check your task list before resubmitting to avoid being billed twice"

func (c *Client) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimRight(c.baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

// execute performs a JSON request with retry-on-transient-failure semantics
// and returns the raw response body plus the final HTTP status code.
func (c *Client) execute(ctx context.Context, method, path string, body interface{}) ([]byte, int, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("tripo3d: failed to marshal request body: %w", err)
		}
	}

	totalAttempts := c.retries + 1
	targetURL := c.buildURL(path)
	idempotent := isIdempotentMethod(method)

	for attempt := 1; attempt <= totalAttempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, c.timeout)

		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(reqCtx, method, targetURL, reader)
		if err != nil {
			cancel()
			return nil, 0, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("User-Agent", c.userAgent)
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			cancel()
			safety := classifyNetErr(ctx, err)
			if canRetry(safety, idempotent) && attempt < totalAttempts {
				time.Sleep(backoff(attempt, ""))
				continue
			}
			return nil, 0, newRequestError(
				fmt.Sprintf("network error: %v", err),
				0, "", err,
				safety == safetyUnknown && !idempotent,
			)
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if readErr != nil {
			// The server responded, so it has already done the work; the
			// failure is purely in reading the reply back.
			if idempotent && attempt < totalAttempts {
				time.Sleep(backoff(attempt, ""))
				continue
			}
			return nil, resp.StatusCode, newRequestError(
				fmt.Sprintf("failed to read response body: %v", readErr),
				resp.StatusCode, "", readErr,
				!idempotent,
			)
		}

		safety := classifyStatus(resp.StatusCode)
		if canRetry(safety, idempotent) && attempt < totalAttempts {
			time.Sleep(backoff(attempt, resp.Header.Get("Retry-After")))
			continue
		}
		if safety == safetyUnknown && !idempotent {
			return nil, resp.StatusCode, newRequestError(
				fmt.Sprintf("HTTP %d", resp.StatusCode),
				resp.StatusCode, string(data), nil,
				true,
			)
		}

		return data, resp.StatusCode, nil
	}

	return nil, 0, &RequestError{Message: "request failed with no response"}
}

// doJSON performs a request and decodes the `data` field of the standard
// envelope into out (which may be nil to discard the payload).
func (c *Client) doJSON(ctx context.Context, method, path string, body, out interface{}) error {
	data, status, err := c.execute(ctx, method, path, body)
	if err != nil {
		return err
	}
	return parseEnvelope(data, status, out)
}

// createTask POSTs params to path and returns the resulting task_id.
func (c *Client) createTask(ctx context.Context, path string, params interface{}) (string, error) {
	var created struct {
		TaskID string `json:"task_id"`
	}
	if err := c.doJSON(ctx, http.MethodPost, path, params, &created); err != nil {
		return "", err
	}
	return created.TaskID, nil
}

// doMultipart uploads a single-part multipart/form-data body.
func (c *Client) doMultipart(ctx context.Context, path, fieldName, filename, contentType string, data []byte, out interface{}) error {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, fieldName, filename))
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		return fmt.Errorf("tripo3d: failed to build multipart body: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return fmt.Errorf("tripo3d: failed to write multipart body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("tripo3d: failed to finalize multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, c.buildURL(path), &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &RequestError{Message: fmt.Sprintf("network error: %v", err), Err: err}
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RequestError{Message: fmt.Sprintf("failed to read response body: %v", err), StatusCode: resp.StatusCode, Err: err}
	}
	return parseEnvelope(respData, resp.StatusCode, out)
}

// getRaw fetches an arbitrary URL without envelope parsing — used to
// download the model files referenced by a completed task's output.
func (c *Client) getRaw(ctx context.Context, rawURL string) ([]byte, string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", &RequestError{Message: fmt.Sprintf("network error: %v", err), Err: err}
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", &RequestError{Message: fmt.Sprintf("failed to read response body: %v", err), StatusCode: resp.StatusCode, Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", &RequestError{
			Message:    fmt.Sprintf("download failed with HTTP %d", resp.StatusCode),
			StatusCode: resp.StatusCode,
			Body:       string(data),
		}
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// parseEnvelope parses the standard `{ code, data, message, suggestion }`
// envelope, decoding `data` into out on success (code == 0), or returning
// *APIError / *RequestError otherwise.
func parseEnvelope(data []byte, statusCode int, out interface{}) error {
	if len(data) == 0 {
		return &RequestError{Message: "empty response body", StatusCode: statusCode}
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return &RequestError{
			Message:    fmt.Sprintf("malformed response: %v", err),
			StatusCode: statusCode,
			Body:       string(data),
		}
	}
	if env.Code != 0 {
		return &APIError{
			Code:       env.Code,
			Message:    env.Message,
			Suggestion: env.Suggestion,
			StatusCode: statusCode,
		}
	}
	if out == nil {
		return nil
	}
	if len(env.Data) == 0 {
		return &RequestError{Message: "response envelope had code=0 but no `data` field", StatusCode: statusCode}
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return &RequestError{
			Message:    fmt.Sprintf("failed to decode data field: %v", err),
			StatusCode: statusCode,
			Body:       string(env.Data),
		}
	}
	return nil
}

// buildPayload merges a typed params struct with its free-form `extra` map,
// producing the final JSON body. This lets every Params struct accept
// forward-compatible fields without per-struct boilerplate.
func buildPayload(v interface{}, extra map[string]interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("tripo3d: failed to marshal params: %w", err)
	}
	m := map[string]interface{}{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("tripo3d: failed to marshal params: %w", err)
	}
	for k, val := range extra {
		m[k] = val
	}
	return m, nil
}

func isIdempotentMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func canRetry(safety retrySafety, idempotent bool) bool {
	switch safety {
	case safetyClean:
		return true
	case safetyUnknown:
		return idempotent
	default:
		return false
	}
}

func classifyStatus(status int) retrySafety {
	switch {
	case cleanRetryStatuses[status]:
		return safetyClean
	case unknownRetryStatuses[status]:
		return safetyUnknown
	default:
		return safetyFatal
	}
}

// classifyNetErr decides how much a transport error tells us about whether
// the request reached the server's handler.
func classifyNetErr(ctx context.Context, err error) retrySafety {
	if ctx.Err() != nil {
		// Caller-supplied context was cancelled/expired — never retry that.
		return safetyFatal
	}

	// Name resolution never produced a connection, and a refused or
	// unreachable peer never accepted one, so the request was never sent.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return safetyClean
	}
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) {
		return safetyClean
	}

	// Everything else — resets, broken pipes, per-attempt timeouts — can
	// happen after the server has already read and acted on the request.
	return safetyUnknown
}

func newRequestError(message string, status int, body string, err error, indeterminate bool) *RequestError {
	if indeterminate {
		message = message + "; " + indeterminateHint
	}
	return &RequestError{
		Message:       message,
		StatusCode:    status,
		Body:          body,
		Err:           err,
		Indeterminate: indeterminate,
	}
}

func backoff(attempt int, retryAfterHeader string) time.Duration {
	if retryAfterHeader != "" {
		if secs, err := strconv.ParseFloat(retryAfterHeader, 64); err == nil && secs >= 0 {
			ms := secs * 1000
			if ms > 30_000 {
				ms = 30_000
			}
			return time.Duration(ms) * time.Millisecond
		}
	}
	baseMs := int64(1000) << (attempt - 1)
	if baseMs > 8000 {
		baseMs = 8000
	}
	jitterMs := rand.Int63n(250)
	return time.Duration(baseMs+jitterMs) * time.Millisecond
}

// pathEscape percent-encodes a single path segment (e.g. a task ID) for
// safe inclusion in a URL.
func pathEscape(segment string) string {
	return url.PathEscape(segment)
}
