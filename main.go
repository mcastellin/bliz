package main

import "github.com/mcastellin/bliz/pkg/cmd"

func main() {
	// this is a test program to measure how the pipelined
	// http requests approach performs with a large number of requests.
	cmd.Execute()
}
