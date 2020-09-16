package process

import (
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
		href := strings.Trim(m[2], " ")
		add := (href != "" && strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "mailto:") && !strings.HasPrefix(href, "javascript")) || strings.HasPrefix(href, site.Root)
		if strings.HasPrefix(href, "/") {
			href = site.Root + href
		}

		_, contains := gatheredUrls.Load(href)
		_, contains2 := gatheredUrls.Load(href + "/")
		_, contains3 := gatheredUrls.Load(strings.TrimSuffix(href, "/"))
		contains = contains || contains2 || contains3

		if !contains {
			gatheredUrls.Store(href, href)
			if add {
				urls = append(urls, data.Link{Referrer: referrer, Url: href})
			}
		}
	}

	return urls
}
