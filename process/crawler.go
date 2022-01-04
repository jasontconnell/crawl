package process

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/jasontconnell/crawl/data"
)

func Start(site *data.Site) {
	gatheredUrls := sync.Map{}
	urls := getStartUrlList(site)

	job := &data.Job{
		Site:       site,
		Urls:       make(chan *data.Link, 1500000),
		Retry:      make(chan *data.Link, 150000),
		Content:    make(chan data.ContentResponse, 300),
		Gathered:   &gatheredUrls,
		Finished:   make(chan bool),
		Processing: true,
	}

	for _, url := range urls {
		job.Urls <- &url
	}

	crawl(job)

	<-job.Finished
}

func printStatus(job *data.Job) {
	if !job.Processing {
		return
	}

	fmt.Printf("\rRoot: %v  Url queue: %d. Retry queue: %d. Content queue: %d. Processed: %d. Errors: %d   \t", job.Site.Root, len(job.Urls), len(job.Retry), len(job.Content), job.Processed, job.ErrorCount)
}

func crawl(job *data.Job) {
	for i := 0; i < runtime.NumCPU(); i++ {
		go getContent(job)
		go getLinks(job)
	}
}

func getStartUrlList(site *data.Site) []data.Link {
	list := []data.Link{}
	list = append(list, data.Link{Url: site.Root})
	for _, i := range site.VirtualPaths {
		list = append(list, data.Link{Url: site.Root + i})
	}
	if site.Sitemap != "" {
		sitemapUrls, err := ReadSitemap(site.Root, site.Sitemap)
		if err != nil {
			log.Fatal("couldn't read sitemap. ", err)
		}
		list = append(list, sitemapUrls...)
	}
	return list
}

func checkDone(job *data.Job) {
	allDone := job.Processed > 0 && len(job.Urls) == 0 && len(job.Content) == 0 && len(job.Retry) == 0
	if allDone {
		job.Processing = false
		job.Finished <- true
	}
}

func getContent(job *data.Job) {
	for {
		select {
		case url := <-job.Urls:
			doGetUrl(job, url)
		case url := <-job.Retry:
			time.Sleep(2 * time.Second)
			url.RetryCount++
			doGetUrl(job, url)
		}

		printStatus(job)
	}
}

func doGetUrl(job *data.Job, url *data.Link) {
	ch, err := getUrlContents(job.Site, *url)
	if err != nil && !errors.Is(err, TimeoutError) {
		job.ErrorCount++
	}

	if errors.Is(err, TimeoutError) {
		if url.RetryCount < job.Site.RetryLimit {
			job.Retry <- url
		}

	}

	job.Content <- ch
}

func getLinks(job *data.Job) {
	for {
		select {
		case cresp := <-job.Content:
			job.Processed++

			if cresp.Code == 200 {
				hrefs := parse(job.Site, cresp.Link.Url, cresp.Content, job.Gathered)
				for _, href := range hrefs {
					job.Urls <- &href
					job.Site.WriteUrl(href.Url, href.Referrer)
				}
			} else if cresp.Code >= 400 {
				msg := "~400 error"
				if cresp.Code >= 500 {
					msg = "~500 error"
				}
				job.Site.WriteError(cresp.Link.Url, cresp.Link.Referrer, cresp.Code, msg)
				job.ErrorCount++
			}

			printStatus(job)
			checkDone(job)
		}
	}
}
