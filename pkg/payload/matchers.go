package payload

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/mcastellin/bliz/pkg/domain"
)

type StatusCodeMatcher struct {
	codes    []int
	matchAll bool
}

func NewStatusCodeMatcher(config string) (*StatusCodeMatcher, error) {
	if config == "all" {
		return &StatusCodeMatcher{matchAll: true}, nil
	}

	bits := strings.Split(config, ",")
	statusCodes := []int{}
	for _, bit := range bits {
		statusRange := strings.Split(bit, "-")
		if len(statusRange) < 2 {
			code, err := strconv.Atoi(bit)
			if err != nil {
				return nil, fmt.Errorf("found invalid status code [%s] for response matcher", bit)
			}
			statusCodes = append(statusCodes, code)
		} else {
			start, err := strconv.Atoi(statusRange[0])
			if err != nil {
				return nil, fmt.Errorf("found invalid status code [%s] for response matcher", statusRange[0])
			}
			end, err := strconv.Atoi(statusRange[1])
			if err != nil {
				return nil, fmt.Errorf("found invalid status code [%s] for response matcher", statusRange[1])
			}
			if start > end {
				return nil, fmt.Errorf("found invalid range [%s] for response matcher", bit)
			}
			for code := start; code <= end; code++ {
				statusCodes = append(statusCodes, code)
			}
		}
	}

	return &StatusCodeMatcher{codes: statusCodes}, nil
}

func (m *StatusCodeMatcher) Match(r domain.FuzzResponse) bool {
	return slices.Contains(m.codes, r.StatusCode)
}
