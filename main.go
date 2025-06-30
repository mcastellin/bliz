package main

import (
	"fmt"
	"time"
)

const baseURL = "localhost:52123"

type reqCounter interface {
	ReqCount() int
}

func findMagic() {
	payloads := make([]string, 100000)
	for i := 0; i < len(payloads); i++ {
		payloads[i] = fmt.Sprintf("%05d", i)
	}

	pipelined := NewPipelinedFuzzer()

	start := time.Now()
	if err := pipelined.Fuzz(baseURL, "/magic/FUZZ.html", payloads); err != nil {
		panic(err)
	}
	elapsed := time.Since(start)
	fmt.Printf("PipelinedFuzzer: took %s\n", elapsed)

	// ===============================================================================
	//defaultFuzzer := NewDefaultFuzzer()

	//start = time.Now()
	//if err := defaultFuzzer.Fuzz(baseURL, "/magic/FUZZ.html", payloads); err != nil {
	//fmt.Printf("%v", err)
	//panic(err)
	//}
	//elapsed = time.Since(start)
	//fmt.Printf("DefaultFuzzer: took %s\n", elapsed)
}

func main() {
	findMagic()
}
