package tripo3d

import (
	"fmt"
	"time"
)

// APIError is a well-formed `{ code, message, suggestion }` error envelope
// with a non-zero `code`, as returned by the Tripo3D API.
type APIError struct {
	Code       int
	Message    string
	Suggestion string
	StatusCode int
}

func (e *APIError) Error() string {
	s := fmt.Sprintf("tripo3d: API error (code=%d)", e.Code)
	if e.Message != "" {
		s += ": " + e.Message
	}
	if e.Suggestion != "" {
		s += " — " + e.Suggestion
	}
	return s
}

// RequestError is a transport-level failure: a network error, a non-2xx
// status without a parseable error envelope, or a malformed response body.
type RequestError struct {
	Message    string
	StatusCode int
	Body       string
	Err        error
}

func (e *RequestError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("tripo3d: request error: %s (HTTP %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("tripo3d: request error: %s", e.Message)
}

func (e *RequestError) Unwrap() error {
	return e.Err
}

// TaskError indicates a task reached a non-successful terminal state
// (failed / cancelled / banned / expired).
type TaskError struct {
	Task *Task
}

func (e *TaskError) Error() string {
	s := fmt.Sprintf("tripo3d: task %s ended with status %q", e.Task.TaskID, e.Task.Status)
	if e.Task.ErrorMsg != "" {
		s += ": " + e.Task.ErrorMsg
	}
	if e.Task.ErrorCode != 0 {
		s += fmt.Sprintf(" (error_code=%d)", e.Task.ErrorCode)
	}
	return s
}

// TimeoutError indicates WaitForTask exceeded the caller-supplied timeout.
type TimeoutError struct {
	TaskID  string
	Timeout time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("tripo3d: timed out after %s waiting for task %s", e.Timeout, e.TaskID)
}
