package custyhttp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type HandlerFunc func(ctx context.Context, r *Request) (*Response, error)

type Request struct {
	Method        string
	Body          []byte
	RequestTarget string
	Protocol      string
	Headers       []string
	Host          string
	UserAgent     string
	Accept        string
	ContentType   string
	ContentLength int
	Raw           []byte
}

type Response struct {
	Body          []byte
	StatusCode    int
	StatusText    string
	Protocol      string
	Headers       []string
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

func BadRequest() *Response {
	return &Response{
		StatusCode: 400,
		StatusText: "Bad Request",
	}
}

func InternalServerError() *Response {
	return &Response{
		StatusCode: 500,
		StatusText: "Internal Server Error",
	}
}

func NotImplemented() *Response {
	return &Response{
		StatusCode: 501,
		StatusText: "Not Implemented",
	}
}

func NotFound() *Response {
	return &Response{
		StatusCode: 404,
		StatusText: "Not Found",
	}
}

func NoContent() *Response {
	return &Response{
		StatusCode: 204,
		StatusText: "No Content",
	}
}

func (h Request) String() string {
	headers := strings.Join(h.Headers, "\r\n")

	req := fmt.Sprintf("%s %s %s\r\n%s\r\n", h.Method, h.RequestTarget, h.Protocol, headers)

	if len(h.Body) == 0 {
		return req + "\r\n"
	}

	return fmt.Sprintf("%s\r\n%s", req, string(h.Body))
}

func (h *Response) String() string {
	headers := strings.Join(h.Headers, "\r\n")

	resp := fmt.Sprintf("%s %d %s\r\n%s\r\n", h.Protocol, h.StatusCode, h.StatusText, headers)

	if len(h.Body) == 0 {
		return resp + "\r\n"
	}

	return fmt.Sprintf("%s\r\n%s", resp, string(h.Body))
}

func (h *Response) populateHeaders() {
	if h.Server != "" {
		h.addHeader("Server", h.Server)
	}
	if h.Date != "" {
		h.addHeader("Date", h.Date)
	}
	if h.CacheControl != "" {
		h.addHeader("Cache-Control", h.CacheControl)
	}
	if h.ContentType != "" {
		h.addHeader("Content-Type", h.ContentType)
	}
	if h.ContentLength != 0 {
		h.addHeader("Content-Length", strconv.Itoa(h.ContentLength))
	}
	if h.Connection != "" {
		h.addHeader("Connection", h.Connection)
	}
	if h.ETag != "" {
		h.addHeader("ETag", h.ETag)
	}
	if h.LastModified != "" {
		h.addHeader("Last-Modified", h.LastModified)
	}
}

func (h *Response) addHeader(key, value string) {
	h.mu.Lock()
	h.Headers = append(h.Headers, fmt.Sprintf("%s: %s", key, value))
	h.mu.Unlock()
}
