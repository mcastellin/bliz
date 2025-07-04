package payload

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type NumericGenerator struct {
	start, end, step int64
	format           string
	nextValue        int64
}

func NewNumericGeneratorS(config string) (*NumericGenerator, error) {
	bits := strings.Split(config, ":")
	var err error
	var start, end, step int64
	var format string
	if start, err = strconv.ParseInt(getOrDefault(bits, 0, "0"), 10, 64); err != nil {
		return nil, err
	}
	if end, err = strconv.ParseInt(getOrDefault(bits, 1, "0"), 10, 64); err != nil {
		return nil, err
	}
	if step, err = strconv.ParseInt(getOrDefault(bits, 2, "1"), 10, 64); err != nil {
		return nil, err
	}
	format = getOrDefault(bits, 3, "%d")

	if start < 0 || end < 0 {
		return nil, fmt.Errorf("negative values are not allowed for numeric generator.")
	}
	if start >= end {
		return nil, fmt.Errorf("invalid sequence for numeric generator.")
	}

	return &NumericGenerator{
		start:     start,
		end:       end,
		step:      step,
		format:    format,
		nextValue: start,
	}, nil
}

func (g *NumericGenerator) Close() error { return nil }

func (g *NumericGenerator) Generate() (string, bool) {
	if g.Done() {
		return g.current(), false
	}
	c := g.current()
	g.nextValue += g.step
	return c, !g.Done()
}
func (g *NumericGenerator) current() string {
	return fmt.Sprintf(g.format, g.nextValue)
}

func (g *NumericGenerator) Done() bool {
	return g.nextValue > g.end
}

func getOrDefault(arr []string, idx int, defaultValue string) string {
	if len(arr) > idx {
		return arr[idx]
	}
	return defaultValue
}

type WordListGenerator struct {
	file    *os.File
	scanner *bufio.Scanner
	done    bool
	current string
}

func NewWordListGenerator(filename string) (*WordListGenerator, error) {
	file, err := os.Open(filename)
	scanner := bufio.NewScanner(file)
	if err != nil {
		return nil, err
	}
	if !scanner.Scan() {
		file.Close()
		return nil, fmt.Errorf("could not initialise file scanner. file is empty.")
	}
	return &WordListGenerator{
		file:    file,
		scanner: scanner,
	}, nil
}

func (g *WordListGenerator) Close() error { return g.file.Close() }

func (g *WordListGenerator) Generate() (string, bool) {
	if g.Done() {
		return g.current, false
	}

	g.current = g.scanner.Text()
	if !g.scanner.Scan() {
		g.done = true
	}
	return g.current, !g.Done()
}
func (g *WordListGenerator) Done() bool {
	return g.done
}
