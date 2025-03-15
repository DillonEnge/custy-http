package custyhttp

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Method string

const (
	METHOD_GET     Method = "GET"
	METHOD_POST    Method = "POST"
	METHOD_PATCH   Method = "PATCH"
	METHOD_PUT     Method = "PUT"
	METHOD_DELETE  Method = "DELETE"
	METHOD_OPTIONS Method = "OPTIONS"
)

type Server struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
	routes       map[string]map[Method]HandlerFunc
	mu           sync.RWMutex
}

func NewServer(opts ...func(*Server)) *Server {
	s := &Server{
		routes: make(map[string]map[Method]HandlerFunc),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func WithReadTimeout(readTimeout time.Duration) func(s *Server) {
	return func(s *Server) {
		s.readTimeout = readTimeout
	}
}

func WithWriteTimeout(writeTimeout time.Duration) func(s *Server) {
	return func(s *Server) {
		s.writeTimeout = writeTimeout
	}
}

func (s *Server) ListenAndServe(ctx context.Context, address string) error {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		if s.readTimeout != 0 {
			conn.SetReadDeadline(time.Now().Add(s.readTimeout))
		}
		if s.writeTimeout != 0 {
			conn.SetWriteDeadline(time.Now().Add(s.readTimeout))
		}

		go func() {
			resp, err := s.handleConn(ctx, conn)
			if err != nil {
				slog.Error("error handling connection", "err", err)
			}

			if resp.ContentLength == 0 && resp.Body != nil && len(resp.Body) > 0 {
				resp.ContentLength = len(resp.Body)
			}
			if resp.Protocol == "" {
				resp.Protocol = "HTTP/1.1"
			}
			if resp.Connection == "" {
				resp.Connection = "close"
			}

			resp.populateHeaders()

			if _, err := conn.Write([]byte(resp.String())); err != nil {
				slog.Error("failed to write to conn", "err", err)
			}
			if err := conn.Close(); err != nil {
				slog.Error("error closing conn", "err", err)
				return
			}
		}()
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) (*Response, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	crlfReplacer := strings.NewReplacer("\r", "", "\n", "")

	reader := bufio.NewReader(conn)

	startLine, err := reader.ReadString('\n')
	if err != nil {
		slog.Error("failed to read startLine", "err", err)
		return BadRequest(), err
	}
	startLineValues := strings.Split(crlfReplacer.Replace(startLine), " ")

	if len(startLineValues) != 3 {
		slog.Error("invalid start line", "startLine", startLine)
		return BadRequest(), err
	}

	httpReq := Request{
		Method:        startLineValues[0],
		RequestTarget: startLineValues[1],
		Protocol:      startLineValues[2],
	}

	s.mu.RLock()
	handlers, ok := s.routes[httpReq.RequestTarget]
	if !ok {
		s.mu.RUnlock()
		return NotFound(), fmt.Errorf("route not found")
	}

	handler, ok := handlers[Method(httpReq.Method)]
	if !ok {
		s.mu.RUnlock()
		return NotImplemented(), fmt.Errorf("method %s not found for route %s", httpReq.Method, httpReq.RequestTarget)
	}
	s.mu.RUnlock()

	// Process headers
	headers := []string{}
	for {
		// t := scanner.Text()
		t, err := reader.ReadString('\n')
		if err != nil {
			slog.Error("failed to read header")
			return BadRequest(), err
		}

		if b, err := reader.Peek(2); string(b) == "\r\n" {
			if err != nil {
				slog.Error("error peeking for end of headers", "err", err)
				return BadRequest(), err
			}
			if _, err := reader.ReadBytes('\n'); err != nil {
				slog.Error("error clearing crlf sequence after headers", "err", err)
				return BadRequest(), err
			}
			break
		}

		fmt.Print(t)

		headerSplit := strings.Split(t, ": ")

		if len(headerSplit) != 2 {
			slog.Error("invalid header", "header", t)
			return BadRequest(), fmt.Errorf("invalid header")
		}

		headerKey, headerValue := headerSplit[0], crlfReplacer.Replace(headerSplit[1])

		switch headerKey {
		case "Content-Type":
			httpReq.ContentType = headerValue
		case "Content-Length":
			httpReq.ContentLength, _ = strconv.Atoi(headerValue)
		case "Host":
			httpReq.Host = headerValue
		case "User-Agent":
			httpReq.UserAgent = headerValue
		case "Accept":
			httpReq.Accept = headerValue
		}

		headers = append(headers, t)
	}

	httpReq.Headers = headers

	if httpReq.ContentLength == 0 {
		slog.Info("no body expected", "httpReq.ContentLength", httpReq.ContentLength)

	}

	contentLength := 0

	for range httpReq.ContentLength {
		// b := scanner.Bytes()
		b, err := reader.ReadByte()
		if err != nil {
			slog.Error("error reading next body byte", "err", err)
			return BadRequest(), err
		}
		contentLength++

		if contentLength > httpReq.ContentLength {
			slog.Error("content length exceeds provided length", "contentLength", contentLength)
			return BadRequest(), fmt.Errorf("content length exceeds provided length")
		}

		httpReq.Body = append(httpReq.Body, b)

		if contentLength == httpReq.ContentLength {
			break
		}
	}

	if contentLength != httpReq.ContentLength {
		slog.Error("mismatched content length", "contentLength", contentLength)
		return BadRequest(), fmt.Errorf("mismatched content length")
	}

	resp, err := handler(ctx, &httpReq)
	if err != nil {
		return InternalServerError(), err
	}

	return resp, nil
}

func (s *Server) registerRoute(method Method, route string, f HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.routes[route]
	if !ok {
		s.routes[route] = map[Method]HandlerFunc{
			method: f,
		}
		return
	}
	s.routes[route][method] = f
}
