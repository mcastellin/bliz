package payload

import (
	"fmt"
	"strings"

	"github.com/mcastellin/turbo-intruder/pkg/domain"
)

type RequestRenderer struct {
}

func NewRequestRenderer() *RequestRenderer {
	return &RequestRenderer{}
}

func (r *RequestRenderer) Render(url, fuzz string) domain.Wrapper {
	var proto, host, path string
	_, after, found := strings.Cut(url, "://")
	var hostAndPath string
	if found {
		hostAndPath = after
	} else {
		hostAndPath = url
	}
	proto = "HTTP/1.1"

	before, after, found := strings.Cut(hostAndPath, "/")
	if found {
		host = before
		path = fmt.Sprintf("/%s", after)
	} else {
		host = before
		path = "/"
	}

	renderedPath := strings.ReplaceAll(path, "FUZZ", fuzz)
	request := fmt.Sprintf("GET %s %s\r\nHost: %s\r\n\r\n", renderedPath, proto, host)

	return domain.Wrapper{
		Host:    host,
		Fuzz:    fuzz,
		Request: request,
	}
}
