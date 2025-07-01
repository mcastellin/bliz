package fuzzer

import (
	"fmt"
	"sync"

	"github.com/mcastellin/turbo-intruder/pkg/domain"
	"github.com/mcastellin/turbo-intruder/pkg/payload"
)

type reqRenderer interface {
	Render(url, fuzz string) domain.Wrapper
}

type PipelinedFuzzer struct {
	batchSize   int
	concurrency int
	reqCount    int
	mu          sync.RWMutex

	renderer reqRenderer
	client   *PooledPipelinedClient
}

func NewPipelinedFuzzer(host string) *PipelinedFuzzer {
	return &PipelinedFuzzer{
		batchSize:   100,
		concurrency: 20,
		renderer:    payload.NewRequestRenderer(),
		client:      NewPooledPipelinedClient(host),
	}
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

func (ff *PipelinedFuzzer) Fuzz(url string, payloads []string) error {

	ff.client.Start()

	go func() {
		defer ff.client.Close()
		for _, fuzz := range payloads {
			w := ff.renderer.Render(url, fuzz)
			ff.client.INC <- w
		}
	}()

	for i := 0; i < len(payloads); i++ {
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
