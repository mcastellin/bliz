package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

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
)

const connectionTimeout = 60

var rootCmd = &cobra.Command{
	Use:   "turbo-intruder",
	Short: "turbo-intruder is a fast fuzzer.",
	Long:  `A fast and flexible http fuzzer built with love`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(targetUrl) == 0 {
			panic(fmt.Errorf("Missing `url` argument."))
		}

		target, err := url.Parse(targetUrl)
		if err != nil {
			panic(err)
		}

		generators := []fuzzer.PayloadGenerator{}

		numericGenerators, _ := cmd.Flags().GetStringArray("gn")
		if len(numericGenerators) > 0 {
			for _, gen := range numericGenerators {
				gen, err := payload.NewNumericPayloadGeneratorS(gen)
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
			BatchSize:           batchSize,
			ClientPoolSize:      threads,
			ConnDeadlineSeconds: connectionTimeout,
		}
		pipelined, err := fuzzer.NewPipelinedFuzzer(fconf, generators)
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
				" :: Matcher          : Response status: 200,204,301,302,307,401,403\n"+
				"________________________________________________\n",
				method,
				targetUrl,
				connectionTimeout,
				threads,
			)

			updateStatus(ui, start, pipelined.ReqCount())
			for {
				select {
				case <-ticker.C:
					updateStatus(ui, start, pipelined.ReqCount())
				case response, more := <-pipelined.OUTC:
					if !more {
						updateStatus(ui, start, pipelined.ReqCount())
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
		if err := pipelined.Fuzz(target, method); err != nil {
			panic(err)
		}
		<-done
	},
}

func updateStatus(ui *termui.TermUI, start time.Time, reqCount int64) {
	elapsed := time.Since(start)
	t := time.Time{}.Add(elapsed)
	ui.UpdateStatus(
		":: Progress: [%d] :: %d req/sec :: Duration: [%s] :: Errors: 0 ::",
		reqCount,
		int(float64(reqCount)/elapsed.Seconds()),
		t.Format("15:04:05"),
	)
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&targetUrl, "url", "u", "", "Target URL")
	rootCmd.PersistentFlags().StringVarP(&method, "method", "X", "GET", "HTTP method to use (default: GET)")

	rootCmd.PersistentFlags().IntVarP(&threads, "threads", "t", 25, "The number of threads used to process requests (default: 25)")
	rootCmd.PersistentFlags().IntVar(&batchSize, "batch-size", 100, "The size of the batch of pipelined requests (default: 100)")

	rootCmd.PersistentFlags().StringArray("gn", []string{}, "Use a numeric generator for fuzzing (value: `start:end:step:format`, example: `0:100:1:%03d`)")
	rootCmd.PersistentFlags().StringArray("gw", []string{}, "Use a wordlist generator for fuzzing (value: `filename`)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
