package fuzzer

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"time"

	"github.com/mcastellin/bliz/pkg/domain"
)

const defaultConnDeadlineSeconds = 60

// Connection handles sending and receiving of HTTP packets
type Connection struct {
	closed bool
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func NewConnection(scheme, host string, timeout time.Duration) (*Connection, error) {
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(time.Duration(defaultConnDeadlineSeconds) * time.Second))

	if scheme == "https" {
		encConn := tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		})
		err := encConn.Handshake()
		if err != nil {
			return nil, fmt.Errorf("TLS handshake error: %w", err)
		}
		conn = encConn
	}
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)
	return &Connection{
		conn:   conn,
		reader: reader,
		writer: writer,
	}, nil
}

func (c *Connection) Send(req *domain.Wrapper) error {
	_, err := c.writer.WriteString(req.Request)
	return err
}

// Read a single request from the connection.
func (c *Connection) Read() (*domain.FuzzResponse, bool, error) {
	resp, err := http.ReadResponse(c.reader, nil)
	if err != nil {
		return nil, false, err
	}
	var body []byte
	more := true
	encoding, ok := resp.Header["Transfer-Encoding"]
	if ok && slices.Contains(encoding, "chunked") {
		body, err = c.readBodyChunked(resp.Body)
	} else if resp.ContentLength >= 0 {
		body, err = c.readBody(resp.ContentLength, resp.Body)
	} else {
		// no content information provided, hence reading the
		// consuming the entire stream for a single request.
		body, err = c.readBodyFull(resp.Body)
		more = false
	}
	if err != nil {
		return nil, false, err
	}

	fr := domain.FuzzResponse{
		Body:       string(body),
		Status:     resp.Status,
		StatusCode: resp.StatusCode,
		Size:       int64(len(body)),
	}

	ka, ok := resp.Header["Connection"]
	if resp.Close || (ok && slices.Contains(ka, "close")) {
		// TCP connection has been closed.
		// We expect all further reads from the conn to fail, hence
		// a new connection is needed to process any more requests
		more = false
	}
	return &fr, more, nil
}

// readBody of HTTP response when Content-Length header is specified.
func (c *Connection) readBody(contentLength int64, stream io.ReadCloser) ([]byte, error) {
	body := make([]byte, contentLength)
	n, _ := stream.Read(body)
	if err := stream.Close(); err != nil {
		return nil, err
	}
	return body[:n], nil
}

func (c *Connection) readBodyChunked(stream io.ReadCloser) ([]byte, error) {

	panic("not implemented")
}

func (c *Connection) readBodyFull(stream io.ReadCloser) ([]byte, error) {
	body, err := io.ReadAll(stream)
	if err != nil {
		return nil, err
	}
	if err := stream.Close(); err != nil {
		return nil, err
	}
	return body, err
}

func (c *Connection) Flush() error {
	return c.writer.Flush()
}

// Close the opened connection
func (c *Connection) Close() error {
	if c.closed {
		return nil
	}
	return c.conn.Close()
}
