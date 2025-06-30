package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type DefaultFuzzer struct {
	reqCount int
	mu       sync.RWMutex
}

func NewDefaultFuzzer() *DefaultFuzzer {
	return &DefaultFuzzer{}
}

func (fz *DefaultFuzzer) addReqCount(num int) {
	fz.mu.Lock()
	defer fz.mu.Unlock()
	fz.reqCount += num
}

func (fz *DefaultFuzzer) ReqCount() int {
	fz.mu.RLocker().Lock()
	defer fz.mu.RLocker().Unlock()
	return fz.reqCount
}

func (fz *DefaultFuzzer) Fuzz(url, path string, payloads []string) error {

	client := &http.Client{}

	for i, item := range payloads {
		renderedPath := strings.ReplaceAll(path, "FUZZ", item)
		resp, err := client.Get(fmt.Sprintf("http://%s%s",
			baseURL,
			renderedPath,
		))
		if err != nil {
			return err
		}
		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			resp.Body.Close()

			fmt.Printf("\n========== Response %s ===========\n", payloads[i])
			fmt.Printf("Status: %s\n", resp.Status)
			fmt.Printf("Content Length: %d\n", resp.ContentLength)
			fmt.Printf("Body: %s\n", body)
		}
		fz.addReqCount(1)
	}
	return nil
}
