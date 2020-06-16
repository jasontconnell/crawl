package data

import (
	"os"
	"sync"
)

type Headers map[string]string

type Site struct {
	Root         string
	VirtualPaths []string
	Headers      Headers
	ErrorFile    *os.File
	UrlsFile     *os.File
}

type Link struct {
	Referrer string
	Url      string
}

type ContentResponse struct {
	Link    Link
	Code    int
	Content string
}

type Job struct {
	Site      Site
	Urls      chan Link
	Content   chan ContentResponse
	Processed int
	Finished  chan bool
	Errors    int
	Gathered  sync.Map
}
