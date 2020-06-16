package process

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/jasontconnell/crawl/data"
	"github.com/pkg/errors"
)

func getUrlContents(site data.Site, link data.Link) (data.ContentResponse, error) {
	cresp := data.ContentResponse{}
	uri, err := url.Parse(link.Url)
	if err != nil {
		return cresp, errors.Wrapf(err, "parsing %s", link.Url)
	}

	headers := make(map[string][]string)
	for k, v := range site.Headers {
		headers[k] = []string{v}
	}

	req := &http.Request{Method: "GET", URL: uri, Header: headers}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if strings.Contains(site.Root, req.URL.Host) {
				return nil
			}
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return cresp, errors.Wrapf(err, "requesting %s", uri)
	}
	defer resp.Body.Close()

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cresp, errors.Wrapf(err, "reading contents from %s", uri)
	}

	cresp.Content = string(contents)
	cresp.Link = link
	cresp.Code = resp.StatusCode

	return cresp, nil
}
