package cmd

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/mcastellin/turbo-intruder/pkg/fuzzer"
	"github.com/mcastellin/turbo-intruder/pkg/payload"
	"github.com/spf13/cobra"
)

var (
	targetUrl string
	method    string
)

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

		pipelined, err := fuzzer.NewPipelinedFuzzer(generators)
		if err != nil {
			panic(err)
		}

		start := time.Now()
		if err := pipelined.Fuzz(target, method); err != nil {
			panic(err)
		}
		elapsed := time.Since(start)
		log.Printf("PipelinedFuzzer: took %s\n", elapsed)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&targetUrl, "url", "u", "", "Target URL")
	rootCmd.PersistentFlags().StringVarP(&method, "method", "X", "GET", "HTTP method to use (default: GET)")

	rootCmd.PersistentFlags().StringArray("gn", []string{}, "Use a numeric generator for fuzzing (value: `start:end:step:format`, example: `0:100:1:%03d`)")
	rootCmd.PersistentFlags().StringArray("gw", []string{}, "Use a wordlist generator for fuzzing (value: `filename`)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
