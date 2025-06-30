package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PipelinedFuzzer struct {
	batchSize int
	reqCount  int
	mu        sync.RWMutex
}

func NewPipelinedFuzzer() *PipelinedFuzzer {
	return &PipelinedFuzzer{
		batchSize: 100,
	}
}

func (fz *PipelinedFuzzer) addReqCount(num int) {
	fz.mu.Lock()
	defer fz.mu.Unlock()
	fz.reqCount += num
}

func (fz *PipelinedFuzzer) ReqCount() int {
	fz.mu.RLocker().Lock()
	defer fz.mu.RLocker().Unlock()
	return fz.reqCount
}

func (fz *PipelinedFuzzer) Fuzz(url, path string, payloads []string) error {

	var err error
	var conn net.Conn
	var writer *bufio.Writer
	var reader *bufio.Reader
	initConn := true

	for i := 0; i < len(payloads); {
		batch := payloads[i:min(len(payloads)-1, i+fz.batchSize)]
		if initConn {
			conn, err = net.Dial("tcp", baseURL)
			if err != nil {
				return err
			}
			defer conn.Close()

			conn.SetDeadline(time.Now().Add(30 * time.Second))
			writer = bufio.NewWriter(conn)
			reader = bufio.NewReader(conn)
			initConn = false
		}

		requests := make([]string, len(batch))
		for i, item := range batch {
			renderedPath := strings.ReplaceAll(path, "FUZZ", item)
			requests[i] = fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\n\r\n", renderedPath, baseURL)
		}

		stop, err := fz.processBatch(writer, reader, requests)
		if err != nil {
			return err
		}

		if stop >= 0 {
			// got disconnected before end of batch
			i += stop
			initConn = true
		} else {
			i += fz.batchSize + 1
		}
	}

	return nil
}

func (fz *PipelinedFuzzer) processBatch(writer *bufio.Writer, reader *bufio.Reader, requests []string) (int, error) {
	for _, req := range requests {
		if _, err := writer.WriteString(req); err != nil {
			return -1, err
		}
	}
	writer.Flush()

	for i := 0; i < len(requests); i++ {
		resp, err := http.ReadResponse(reader, nil)
		if err != nil {
			return -1, err
		}

		body := make([]byte, resp.ContentLength)
		n, _ := resp.Body.Read(body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			fmt.Printf("\n========== Response %s ===========\n", "XXX")
			fmt.Printf("Status: %s\n", resp.Status)
			fmt.Printf("Content Length: %d\n", resp.ContentLength)
			fmt.Printf("Body: %s\n", body[:n])
		}

		fz.addReqCount(1)

		connHeader := resp.Header["Connection"]
		if len(connHeader) == 0 || connHeader[0] != "keep-alive" {
			// TCP connection has been closed. We expect any further request to fail.
			return i, nil
		}
	}
	return -1, nil
}
