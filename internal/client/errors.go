package client

import (
	"fmt"
	"io"
	"net/http"
)

type ErrorResponse struct {
	StatusCode int
	Body       string
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("cloudfly api error: status %d: %s", e.StatusCode, e.Body)
}

func AsError(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return &ErrorResponse{StatusCode: resp.StatusCode, Body: string(body)}
}
