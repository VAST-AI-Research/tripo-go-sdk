package tripo3d

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var defaultRetryStatuses = map[int]bool{
	408: true, 425: true, 429: true, 500: true, 502: true, 503: true, 504: true,
}

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
			if attempt < totalAttempts && isRetryableNetErr(ctx, err) {
				time.Sleep(backoff(attempt, ""))
				continue
			}
			return nil, 0, &RequestError{Message: fmt.Sprintf("network error: %v", err), Err: err}
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if readErr != nil {
			if attempt < totalAttempts {
				time.Sleep(backoff(attempt, ""))
				continue
			}
			return nil, resp.StatusCode, &RequestError{
				Message:    fmt.Sprintf("failed to read response body: %v", readErr),
				StatusCode: resp.StatusCode,
				Err:        readErr,
			}
		}

		if defaultRetryStatuses[resp.StatusCode] && attempt < totalAttempts {
			time.Sleep(backoff(attempt, resp.Header.Get("Retry-After")))
			continue
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

func isRetryableNetErr(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		// Caller-supplied context was cancelled/expired — never retry that.
		return false
	}
	return true
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
