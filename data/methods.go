package data

import (
	"fmt"
	"net/url"
	"os"
)

func NewSite(root string, virtualPaths []string, headers Headers, urlFilename, errorFilename string, timeout, retryLimit int) (*Site, error) {
	site := new(Site)
	site.Root = root
	site.Timeout = timeout
	site.RetryLimit = retryLimit
	u, err := url.Parse(root)
	if err != nil {
		return nil, fmt.Errorf("parsing url %s. %w", root, err)
	}

	site.RootUrl = u
	site.VirtualPaths = virtualPaths
	site.Headers = headers

	ef, err := os.OpenFile(errorFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("opening error file: %w", err)
	}

	uf, err := os.OpenFile(urlFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("opening error file: %w", err)
	}

	site.errorFile = ef
	site.urlsFile = uf

	site.errorFile.WriteString("Url, Referrer, Code, Details\n")
	site.urlsFile.WriteString("Url, Referrer\n")

	return site, nil
}

func (s *Site) CleanUp() {
	s.errorFile.Close()
	s.urlsFile.Close()
}

func (s Site) WriteError(url, referrer string, code int, details string) {
	s.errorFile.WriteString(fmt.Sprintf("%s, %s, %d, %s\n", url, referrer, code, details))
}

func (s Site) WriteUrl(url, referrer string) {
	s.urlsFile.WriteString(fmt.Sprintf("%s, %s\n", url, referrer))
}
