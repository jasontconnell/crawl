package process

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/jasontconnell/crawl/data"
)

type sitemapXml struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []sitemapUrl `xml:"url"`
}

type sitemapUrl struct {
	XMLName  xml.Name `xml:"url"`
	Location string   `xml:"loc"`
}

func ReadSitemap(root, sitemap string) ([]data.Link, error) {
	u := root + sitemap
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("can't create request %s %w", u, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error on request. %s %w", u, err)
	}
	defer resp.Body.Close()

	dec := xml.NewDecoder(resp.Body)

	sm := sitemapXml{}
	err = dec.Decode(&sm)
	if err != nil {
		return nil, fmt.Errorf("decoding sitemap xml %w", err)
	}

	links := []data.Link{}
	for _, item := range sm.URLs {
		link := data.Link{Url: item.Location, Referrer: sitemap}
		links = append(links, link)
	}
	return links, nil
}
