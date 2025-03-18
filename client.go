package custyhttp

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

type Client struct {
	baseURL string
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
	}
}

func (c *Client) Do(method Method, req *Request) (*Response, error) {
	if req.RequestTarget == "" {
		return nil, fmt.Errorf("missing request target")
	}
	if req.Accept == "" {
		req.Accept = "*/*"
	}
	if req.Protocol == "" {
		req.Protocol = "HTTP/1.1"
	}

	req.Method = string(method)

	conn, err := net.Dial("tcp", c.baseURL)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Error("failed to close conn", "err", err)
		}
	}()

	if req.ContentLength != 0 && req.Body() != nil && len(req.Body()) > 0 {
		req.ContentLength = len(req.Body())
	}

	req.Host = conn.LocalAddr().String()

	if _, err := conn.Write([]byte(req.String())); err != nil {
		return nil, err
	}

	resp, err := c.parseResponse(conn)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) parseResponse(conn net.Conn) (*Response, error) {
	crlfReplacer := strings.NewReplacer("\r", "", "\n", "")

	reader := bufio.NewReader(conn)

	startLine, err := reader.ReadString('\n')
	if err != nil {
		slog.Error("failed to read startLine", "err", err)
		return nil, err
	}
	startLineValues := strings.SplitN(crlfReplacer.Replace(startLine), " ", 3)

	if len(startLineValues) != 3 {
		slog.Error("invalid start line", "startLine", startLine)
		return nil, err
	}

	statusCode, err := strconv.Atoi(startLineValues[1])
	if err != nil {
		return nil, err
	}

	resp := Response{
		Protocol:   startLineValues[0],
		StatusCode: statusCode,
		StatusText: startLineValues[2],
	}

	// Process headers
	headers := []string{}
	for {
		// t := scanner.Text()
		t, err := reader.ReadString('\n')
		if err != nil {
			slog.Error("failed to read header")
			return nil, err
		}

		if b, err := reader.Peek(2); string(b) == "\r\n" {
			if err != nil {
				slog.Error("error peeking for end of headers", "err", err)
				return nil, err
			}
			if _, err := reader.ReadBytes('\n'); err != nil {
				slog.Error("error clearing crlf sequence after headers", "err", err)
				return nil, err
			}
			break
		}

		headerSplit := strings.SplitN(t, ": ", 2)

		if len(headerSplit) != 2 {
			slog.Error("invalid header", "header", t)
			return nil, fmt.Errorf("invalid header")
		}

		headerKey, headerValue := headerSplit[0], crlfReplacer.Replace(headerSplit[1])

		switch strings.ToLower(headerKey) {
		case "content-type":
			resp.ContentType = headerValue
		case "content-length":
			resp.ContentLength, _ = strconv.Atoi(headerValue)
		case "cache-control":
			resp.CacheControl = headerValue
		case "date":
			resp.Date = headerValue
		case "server":
			resp.Server = headerValue
		}

		headers = append(headers, t)
	}

	resp.SetHeaders(headers)

	if resp.ContentLength == 0 {
		slog.Info("no body expected", "resp.ContentLength", resp.ContentLength)
	}

	contentLength := 0

	for range resp.ContentLength {
		b, err := reader.ReadByte()
		if err != nil {
			slog.Error("error reading next body byte", "err", err)
			return nil, err
		}
		contentLength++

		if contentLength > resp.ContentLength {
			slog.Error("content length exceeds provided length", "contentLength", contentLength)
			return nil, fmt.Errorf("content length exceeds provided length")
		}

		resp.WriteBytesToBody([]byte{b})

		if contentLength == resp.ContentLength {
			break
		}
	}

	if contentLength != resp.ContentLength {
		slog.Error("mismatched content length", "contentLength", contentLength)
		return nil, fmt.Errorf("mismatched content length")
	}

	return &resp, nil
}
