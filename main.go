package main

import (
	"fmt"
	"log"
	"time"

	"github.com/mcastellin/turbo-intruder/pkg/fuzzer"
)

const baseURL = "localhost:52124"

type reqCounter interface {
	ReqCount() int
}

func findMagicNumber() {
	payloads := make([]string, 100000)
	for i := 0; i < len(payloads); i++ {
		payloads[i] = fmt.Sprintf("%05d", i)
	}

	pipelined := fuzzer.NewPipelinedFuzzer(baseURL)

	start := time.Now()
	url := fmt.Sprintf("http://%s%s", baseURL, "/magic/FUZZ.html")
	if err := pipelined.Fuzz(url, payloads); err != nil {
		panic(err)
	}
	elapsed := time.Since(start)
	log.Printf("PipelinedFuzzer: took %s\n", elapsed)
}

func main() {
	// this is a test program to measure how the pipelined
	// http requests approach performs with a large number of requests.
	findMagicNumber()
}
