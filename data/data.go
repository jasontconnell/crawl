package data

import (
	"net/url"
	"os"
	"sync"
)

type Headers map[string]string

type Site struct {
	Root         string
	RootUrl      *url.URL
	Sitemap      string
	VirtualPaths []string
	ReplaceRoots []string
	Headers      Headers
	Timeout      int
	RetryLimit   int
	errorFile    *os.File
	urlsFile     *os.File
}

type Link struct {
	Referrer   string
	Url        string
	RetryCount int
}

type ContentResponse struct {
	Link    Link
	Code    int
	Content string
	Retry   bool
}

type Job struct {
	Site       *Site
	Urls       chan *Link
	Retry      chan *Link
	Content    chan ContentResponse
	Processed  int
	Finished   chan bool
	ErrorCount int
	Gathered   *sync.Map
	Processing bool
}
