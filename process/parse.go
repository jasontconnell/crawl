package process

import (
	"net/url"
	"regexp"
	"strings"
	"sync"

	"github.com/jasontconnell/crawl/data"
)

var hrefRegex *regexp.Regexp = regexp.MustCompile(`(href|src)="(.*?)"`)

func parse(site *data.Site, referrer, content string, gatheredUrls *sync.Map) []data.Link {
	urls := []data.Link{}
	matches := hrefRegex.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		u, err := url.Parse(m[2])
		if err != nil {
			continue
		}

		add := (!u.IsAbs() || u.Hostname() == site.RootUrl.Hostname()) && !strings.HasPrefix(u.String(), "#")
		if !add {
			continue
		}

		if !u.IsAbs() {
			u = site.RootUrl.ResolveReference(u)
		}

		_, contains := gatheredUrls.Load(u.String())
		// _, contains2 := gatheredUrls.Load(href + "/")
		// _, contains3 := gatheredUrls.Load(strings.TrimSuffix(href, "/"))
		// contains = contains || contains2 || contains3

		if !contains {
			gatheredUrls.Store(u.String(), u.String())
			if add {
				urls = append(urls, data.Link{Referrer: referrer, Url: u.String()})
			}
		}
	}

	return urls
}
