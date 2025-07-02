package fuzzer

import (
	"fmt"
	"net/url"
	"strings"
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
type ResponseMatcher interface {
	Match(domain.FuzzResponse) bool
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
	reqCount int64
	mu       sync.RWMutex

	renderer   reqRenderer
	generators []PayloadGenerator
	matchers   []ResponseMatcher
	client     *PooledPipelinedClient
	OUTC       chan domain.FuzzResponse
}

func NewPipelinedFuzzer(c Config, generators []PayloadGenerator, matchers []ResponseMatcher) (*PipelinedFuzzer, error) {
	if len(generators) == 0 {
		return nil, fmt.Errorf("could not find valid payload generators for fuzzer.")
	}
	return &PipelinedFuzzer{
		renderer:   payload.NewRequestRenderer(),
		generators: generators,
		matchers:   matchers,
		client:     NewPooledPipelinedClient(c),
		OUTC:       make(chan domain.FuzzResponse, 10),
	}, nil
}

func (ff *PipelinedFuzzer) addReqCount(num int64) {
	ff.mu.Lock()
	defer ff.mu.Unlock()
	ff.reqCount += num
}

func (ff *PipelinedFuzzer) ReqCount() int64 {
	ff.mu.RLocker().Lock()
	defer ff.mu.RLocker().Unlock()
	return ff.reqCount
}

func (ff *PipelinedFuzzer) Fuzz(targetURL *url.URL, method string) error {
	defer close(ff.OUTC)

	ff.client.Start()
	var mu sync.RWMutex
	var generated int64 = 0
	doneGenerating := false
	completed := false

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
		completed = doneGenerating && generated == ff.ReqCount()
		mu.RUnlock()
		if completed {
			break
		}
		response, more := <-ff.client.OUTC
		ff.addReqCount(1)
		response.Lines = strings.Count(response.Body, "\n")
		response.Words = len(strings.Fields(response.Body))
		if !more {
			return fmt.Errorf("channel was closed but there were more messages to process")
		}
		for _, matcher := range ff.matchers {
			if matcher.Match(response) {
				ff.OUTC <- response
			}
		}
	}

	return nil
}
