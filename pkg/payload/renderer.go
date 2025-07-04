package payload

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mcastellin/bliz/pkg/domain"
)

type RequestRenderer struct {
	target *url.URL
	method string
}

func NewRequestRenderer(target *url.URL, method string) *RequestRenderer {
	return &RequestRenderer{
		target: target,
		method: method,
	}
}

func (r *RequestRenderer) URL() string    { return r.target.String() }
func (r *RequestRenderer) Method() string { return r.method }

func (r *RequestRenderer) Render(fuzz []string) domain.Wrapper {
	req := []string{
		fmt.Sprintf("%s %s %s", r.method, r.target.Path, "HTTP/1.2"),
		fmt.Sprintf("Host: %s", r.target.Host),
		// todo: check if there is any way we can negotiate a keep-alive
		// from the http request.
		//fmt.Sprintf("Connection: %s", "keep-alive"),
		"\r\n",
	}
	request := strings.Join(req, "\r\n")

	return domain.Wrapper{
		Scheme:  r.target.Scheme,
		Host:    fmt.Sprintf("%s:%s", r.target.Hostname(), getPort(r.target.Port(), r.target.Scheme)),
		Fuzz:    fuzz,
		Request: fuzzRequest(request, fuzz),
	}
}

type RawRequestRenderer struct {
	scheme                    string
	host                      string
	method, path, httpVersion string
	template                  string
}

func NewRawRequestRenderer(reqTemplate, scheme string) (*RawRequestRenderer, error) {
	// make sure every new line character is represented with `\r\n`
	safeTemplate := strings.ReplaceAll(
		strings.ReplaceAll(reqTemplate, "\r\n", "\n"),
		"\n", "\r\n",
	)
	host := getHostFromRequestTemplate(safeTemplate)
	if host == nil {
		return nil, fmt.Errorf("could not extract a valid host from request template")
	}

	method, path, version := getFieldsFromRequestTemplate(safeTemplate)
	return &RawRequestRenderer{
		scheme:      scheme,
		host:        *host,
		method:      method,
		path:        path,
		httpVersion: version,
		template:    safeTemplate,
	}, nil
}

func (r *RawRequestRenderer) URL() string    { return fmt.Sprintf("%s://%s%s", r.scheme, r.host, r.path) }
func (r *RawRequestRenderer) Method() string { return r.method }

func (r *RawRequestRenderer) Render(fuzz []string) domain.Wrapper {

	hostAndPort := r.host
	if strings.Index(hostAndPort, ":") < 0 {
		hostAndPort = fmt.Sprintf("%s:%s", hostAndPort, getPort("", r.scheme))
	}
	return domain.Wrapper{
		Scheme:  r.scheme,
		Host:    hostAndPort,
		Fuzz:    fuzz,
		Request: fuzzRequest(r.template, fuzz),
	}
}

func fuzzRequest(request string, fuzz []string) string {
	var renderedRequest string
	if len(fuzz) == 0 {
		panic("no fuzz could be found to render payload")
	}
	if len(fuzz) == 1 {
		renderedRequest = strings.ReplaceAll(request, "FUZZ", fuzz[0])
	} else {
		renderedRequest = request
		for _, replacement := range fuzz {
			renderedRequest = strings.Replace(request, "FUZZ", replacement, 1)
		}
	}
	return renderedRequest
}

func getFieldsFromRequestTemplate(template string) (string, string, string) {
	var pathLine string
	idx := strings.Index(template, "\r\n")
	if idx > 0 {
		pathLine = template[:idx]
	} else {
		pathLine = template
	}
	f := strings.Fields(pathLine)
	if len(f) < 3 {
		panic("could not read request fields from template")
	}
	return f[0], f[1], f[2]
}

func getHostFromRequestTemplate(template string) *string {
	idx := strings.Index(template, "Host:")
	if idx < 0 {
		return nil
	}
	hostLine := template[idx:]
	idx = strings.Index(hostLine, "\r\n")
	if idx > 0 {
		hostLine = hostLine[:idx]
	}

	host := strings.TrimSpace(strings.ReplaceAll(hostLine, "Host:", ""))
	return &host
}

func getPort(port, scheme string) string {
	if len(port) > 0 {
		return port
	}

	if scheme == "http" {
		return "80"
	} else if scheme == "https" {
		return "443"
	}
	return port
}
