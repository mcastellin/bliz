package fuzzer

import (
	"fmt"
	"net/url"
	"sync"

	"github.com/mcastellin/turbo-intruder/pkg/domain"
	"github.com/mcastellin/turbo-intruder/pkg/payload"
)

type reqRenderer interface {
	Render(url *url.URL, method string, fuzz []string) domain.Wrapper
}
type PayloadGenerator interface {
	Generate() (string, bool)
	Close() error
}

func getPayload(generators []PayloadGenerator) ([]string, bool) {
	fuzz := make([]string, len(generators))
	more := true

	for i := 0; i < len(fuzz); i++ {
		val, hasMore := generators[i].Generate()
		more = more && hasMore
		fuzz[i] = val
	}
	return fuzz, more
}

type PipelinedFuzzer struct {
	batchSize   int
	concurrency int
	reqCount    int
	mu          sync.RWMutex

	renderer   reqRenderer
	generators []PayloadGenerator
	client     *PooledPipelinedClient
}

func NewPipelinedFuzzer(generators []PayloadGenerator) (*PipelinedFuzzer, error) {
	if len(generators) == 0 {
		return nil, fmt.Errorf("could not find valid payload generators for fuzzer.")
	}
	return &PipelinedFuzzer{
		batchSize:   100,
		concurrency: 20,
		renderer:    payload.NewRequestRenderer(),
		generators:  generators,
		client:      NewPooledPipelinedClient(),
	}, nil
}

func (ff *PipelinedFuzzer) addReqCount(num int) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.reqCount += num
}

func (ff *PipelinedFuzzer) ReqCount() int {
	ff.mu.RLocker().Lock()
	defer ff.mu.RLocker().Unlock()
	return ff.reqCount
}

func (ff *PipelinedFuzzer) Fuzz(targetURL *url.URL, method string) error {

	ff.client.Start()
	var mu sync.RWMutex
	generated := 0
	doneGenerating := false

	go func() {
		defer ff.client.Close()
		for {
			fuzz, more := getPayload(ff.generators)
			w := ff.renderer.Render(targetURL, method, fuzz)
			ff.client.INC <- w
			mu.Lock()
			generated += 1
			mu.Unlock()

			if !more {
				doneGenerating = true
				return
			}
		}
	}()

	for {
		mu.RLock()
		completed := doneGenerating && generated == ff.ReqCount()
		mu.RUnlock()
		if completed {
			break
		}
		response, more := <-ff.client.OUTC
		if !more {
			return fmt.Errorf("channel was closed but there were more messages to process")
		}
		if response.StatusCode >= 200 && response.StatusCode <= 299 {
			fmt.Printf("\n========== Response %s ===========\n", response.Req.Fuzz)
			fmt.Printf("Status: %s\n", response.Status)
			fmt.Printf("Body: %s\n", response.Body)
		}
		ff.addReqCount(1)
	}

	return nil
}
