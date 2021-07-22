package process

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jasontconnell/crawl/data"
)

var TimeoutError error = errors.New("timeout")

func getUrlContents(site *data.Site, link data.Link) (data.ContentResponse, error) {
	cresp := data.ContentResponse{}
	uri, err := url.Parse(link.Url)
	if err != nil {
		return cresp, fmt.Errorf("parsing %s. %w", link.Url, err)
	}

	headers := make(map[string][]string)
	for k, v := range site.Headers {
		headers[k] = []string{v}
	}

	req := &http.Request{Method: "GET", URL: uri, Header: headers}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if strings.Contains(site.Root, req.URL.Hostname()) {
				return nil
			}
			return http.ErrUseLastResponse
		},
		Timeout: time.Duration(time.Second * 5),
	}

	resp, err := client.Do(req)
	if err != nil {
		if toerr, ok := err.(net.Error); ok && toerr.Timeout() {
			site.WriteError(link.Url, link.Referrer, -1, "timed out")
			return cresp, TimeoutError
		}

		site.WriteError(link.Url, link.Referrer, -1, err.Error())
		return cresp, fmt.Errorf("requesting %s : %w", uri, err)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cresp, fmt.Errorf("reading contents from %s. %w", uri, err)
	}

	cresp.Content = string(contents)
	cresp.Link = link
	cresp.Code = resp.StatusCode

	return cresp, nil
}
