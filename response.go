package custyhttp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type HandlerFunc func(ctx context.Context, r *Request) (*Response, error)

type Response struct {
	body          []byte
	StatusCode    int
	StatusText    string
	Protocol      string
	headers       []string
	Server        string
	Date          string
	CacheControl  string
	ContentType   string
	ContentLength int
	Connection    string
	ETag          string
	LastModified  string
	mu            sync.Mutex
}

func badRequest() *Response {
	return &Response{
		StatusCode: 400,
		StatusText: "Bad Request",
	}
}

func internalServerError() *Response {
	return &Response{
		StatusCode: 500,
		StatusText: "Internal Server Error",
	}
}

func notImplemented() *Response {
	return &Response{
		StatusCode: 501,
		StatusText: "Not Implemented",
	}
}

func notFound() *Response {
	return &Response{
		StatusCode: 404,
		StatusText: "Not Found",
	}
}

func noContent() *Response {
	return &Response{
		StatusCode: 204,
		StatusText: "No Content",
	}
}

func (r *Response) String() string {
	headers := strings.Join(r.Headers(), "\r\n")

	resp := fmt.Sprintf("%s %d %s\r\n%s\r\n", r.Protocol, r.StatusCode, r.StatusText, headers)

	if len(r.Body()) == 0 {
		return resp + "\r\n"
	}

	return fmt.Sprintf("%s\r\n%s", resp, string(r.Body()))
}

func (r *Response) Body() []byte {
	r.mu.Lock()
	body := make([]byte, len(r.body))
	copy(body, r.body)
	r.mu.Unlock()

	return body
}

func (r *Response) Headers() []string {
	r.mu.Lock()
	headers := make([]string, len(r.headers))
	copy(headers, r.headers)
	r.mu.Unlock()

	return headers
}

func (r *Response) SetHeaders(headers []string) {
	r.mu.Lock()
	r.headers = headers
	r.mu.Unlock()
}

func (r *Response) populateHeaders() {
	if r.Server != "" {
		r.addHeader("Server", r.Server)
	}
	if r.Date != "" {
		r.addHeader("Date", r.Date)
	}
	if r.CacheControl != "" {
		r.addHeader("Cache-Control", r.CacheControl)
	}
	if r.ContentType != "" {
		r.addHeader("Content-Type", r.ContentType)
	}
	if r.ContentLength != 0 {
		r.addHeader("Content-Length", strconv.Itoa(r.ContentLength))
	}
	if r.Connection != "" {
		r.addHeader("Connection", r.Connection)
	}
	if r.ETag != "" {
		r.addHeader("ETag", r.ETag)
	}
	if r.LastModified != "" {
		r.addHeader("Last-Modified", r.LastModified)
	}
}

func (r *Response) addHeader(key, value string) {
	r.mu.Lock()
	r.headers = append(r.headers, fmt.Sprintf("%s: %s", key, value))
	r.mu.Unlock()
}

func (r *Response) WriteBytesToBody(b []byte) {
	r.mu.Lock()
	r.body = append(r.body, b...)
	r.mu.Unlock()
}
