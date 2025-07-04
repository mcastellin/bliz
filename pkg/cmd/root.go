package cmd

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mcastellin/turbo-intruder/pkg/domain"
	"github.com/mcastellin/turbo-intruder/pkg/fuzzer"
	"github.com/mcastellin/turbo-intruder/pkg/payload"
	termui "github.com/mcastellin/turbo-intruder/pkg/ui"
	"github.com/spf13/cobra"
)

var (
	targetUrl string
	method    string
	threads   int
	batchSize int

	connectionTimeout int
	matchCodes        string

	requestTemplateFile string
	requestScheme       string
)

type fuzzStats interface {
	ReqCount() int64
	ConnCreateCount() int64
}

type reqRenderer interface {
	Render(fuzz []string) domain.Wrapper
	URL() string
	Method() string
}

var rootCmd = &cobra.Command{
	Use:   "turbo-intruder",
	Short: "turbo-intruder is a fast fuzzer.",
	Long:  `A fast and flexible http fuzzer built with love`,
	Run: func(cmd *cobra.Command, args []string) {

		var requestRenderer reqRenderer
		var err error
		if len(requestTemplateFile) > 0 {
			var requestTemplate []byte
			if requestTemplateFile == "-" {
				requestTemplate, err = io.ReadAll(os.Stdin)
				if err != nil {
					panic(fmt.Errorf("error reading request template from stdin: %w", err))
				}
			} else {
				requestTemplate, err = os.ReadFile(requestTemplateFile)
				if err != nil {
					panic(fmt.Errorf("error reading file from path %s: %w", requestTemplateFile, err))
				}
			}
			requestRenderer, err = payload.NewRawRequestRenderer(string(requestTemplate), requestScheme)
			if err != nil {
				panic(err)
			}
		} else {
			if len(targetUrl) == 0 {
				panic(fmt.Errorf("Missing `url` argument."))
			}

			target, err := url.Parse(targetUrl)
			if err != nil {
				panic(err)
			}
			requestRenderer = payload.NewRequestRenderer(target, method)
		}

		generators := []fuzzer.PayloadGenerator{}

		numericGenerators, _ := cmd.Flags().GetStringArray("gn")
		if len(numericGenerators) > 0 {
			for _, gen := range numericGenerators {
				gen, err := payload.NewNumericGeneratorS(gen)
				if err != nil {
					panic(err)
				}
				generators = append(generators, gen)
				defer gen.Close()
			}
		}

		wordlistGenerators, _ := cmd.Flags().GetStringArray("gw")
		if len(wordlistGenerators) > 0 {
			for _, gen := range wordlistGenerators {
				gen, err := payload.NewWordListGenerator(gen)
				if err != nil {
					panic(err)
				}
				generators = append(generators, gen)
				defer gen.Close()
			}
		}

		fconf := fuzzer.Config{
			BatchSize:          batchSize,
			ClientPoolSize:     threads,
			DialTimeoutSeconds: connectionTimeout,
		}

		statusMatcher, err := payload.NewStatusCodeMatcher(matchCodes)
		if err != nil {
			panic(err)
		}
		pipelined, err := fuzzer.NewPipelinedFuzzer(
			fconf,
			requestRenderer,
			generators,
			[]fuzzer.ResponseMatcher{statusMatcher},
		)
		if err != nil {
			panic(err)
		}

		done := make(chan struct{})
		go func() {
			start := time.Now()
			ui := &termui.TermUI{}

			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			ui.Printf("\n________________________________________________\n\n"+
				" :: Method           : %s\n"+
				" :: URL              : %s\n"+
				" :: Follow redirects : false\n"+
				" :: Pipelining       : true\n"+
				" :: Conn Timeout     : %d seconds\n"+
				" :: Threads          : %d\n"+
				" :: Matcher          : Response status: %s\n"+
				"________________________________________________\n",
				requestRenderer.Method(),
				requestRenderer.URL(),
				connectionTimeout,
				threads,
				matchCodes,
			)

			updateStatus(ui, start, pipelined)
			for {
				select {
				case <-ticker.C:
					updateStatus(ui, start, pipelined)
				case response, more := <-pipelined.OUTC:
					if !more {
						ui.Printf("\n")
						updateStatus(ui, start, pipelined)
						close(done)
						return
					}
					ui.Printf("%-30s [Status: %d, Size: %d, Words: %d, Lines: %d]\n",
						strings.Join(response.Req.Fuzz, ","),
						response.StatusCode,
						response.Size,
						response.Words,
						response.Lines,
					)
				}
			}
		}()
		if err := pipelined.Fuzz(); err != nil {
			panic(err)
		}
		<-done
	},
}

func updateStatus(ui *termui.TermUI, start time.Time, stats fuzzStats) {
	elapsed := time.Since(start)
	t := time.Time{}.Add(elapsed)
	ui.UpdateStatus(
		":: Progress: [%d] :: %d req/sec :: Duration: [%s] :: Errors: 0 :: Total conns: %d ::",
		stats.ReqCount(),
		int(float64(stats.ReqCount())/elapsed.Seconds()),
		t.Format("15:04:05"),
		stats.ConnCreateCount(),
	)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&targetUrl, "url", "u", "", "Target URL")
	rootCmd.PersistentFlags().StringVarP(&method, "method", "X", "GET", "HTTP method to use (default: GET)")

	rootCmd.PersistentFlags().IntVarP(&threads, "threads", "t", 25, "The number of threads used to process requests (default: 25)")
	rootCmd.PersistentFlags().IntVar(&batchSize, "batch-size", 100, "The size of the batch of pipelined requests (default: 100)")
	rootCmd.PersistentFlags().IntVar(&connectionTimeout, "timeout", 10, "The connection timeout in seconds (default: 10)")

	rootCmd.PersistentFlags().StringArrayP("gw", "w", []string{}, "Use a wordlist generator for fuzzing (value: `filename`)")
	rootCmd.PersistentFlags().StringArray("gn", []string{}, "Use a numeric generator for fuzzing (value: `start:end:step:format`, example: `0:100:1:%03d`)")

	rootCmd.PersistentFlags().StringVar(&requestTemplateFile, "request", "", "Use request template from file. Use '-' to read template from STDIN (value: `filename`)")
	rootCmd.PersistentFlags().StringVar(&requestScheme, "request-scheme", "https", "Specify the protocol scheme to use with a request template (value `scheme`, default: `https`)")

	rootCmd.PersistentFlags().StringVar(&matchCodes, "mc", "200,204,301,302,307,401,403",
		"Match HTTP status codes, or 'all' for everything. (value: `httpStatus`, default: 200,204,301,302,307,401,403)")

}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
