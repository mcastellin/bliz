package payload

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mcastellin/turbo-intruder/pkg/domain"
)

type RequestRenderer struct {
}

func NewRequestRenderer() *RequestRenderer {
	return &RequestRenderer{}
}

func (r *RequestRenderer) Render(targetURL *url.URL, method string, fuzz []string) domain.Wrapper {

	var renderedPath string
	if len(fuzz) == 0 {
		panic("no fuzz could be found to render payload")
	}
	if len(fuzz) == 1 {
		renderedPath = strings.ReplaceAll(targetURL.Path, "FUZZ", fuzz[0])
	} else {
		renderedPath = targetURL.Path
		for _, replacement := range fuzz {
			renderedPath = strings.Replace(renderedPath, "FUZZ", replacement, 1)
		}
	}

	req := []string{
		fmt.Sprintf("%s %s %s", method, renderedPath, "HTTP/1.2"),
		fmt.Sprintf("Host: %s", targetURL.Host),
		// todo: check if theres is any way we can negtiate a keep-alive
		// from the http request.
		//fmt.Sprintf("Connection: %s", "keep-alive"),
		"\r\n",
	}

	return domain.Wrapper{
		Scheme:  targetURL.Scheme,
		Host:    fmt.Sprintf("%s:%s", targetURL.Hostname(), getPort(targetURL)),
		Fuzz:    fuzz,
		Request: strings.Join(req, "\r\n"),
	}
}

func getPort(u *url.URL) string {
	portString := u.Port()
	if len(portString) > 0 {
		return portString
	}

	if u.Scheme == "http" {
		return "80"
	} else if u.Scheme == "https" {
		return "443"
	}
	return u.Port()
}
