package process

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"

	"github.com/jasontconnell/crawl/data"
)

func Start(site data.Site) {
	gatheredUrls := sync.Map{}
	urls := getStartUrlList(site)
	churl := make(chan data.Link, 1500000)
	chcresp := make(chan data.ContentResponse, 300)
	finished := make(chan bool)

	job := &data.Job{
		Site:     site,
		Urls:     churl,
		Content:  chcresp,
		Gathered: gatheredUrls,
		Finished: finished,
	}

	for _, url := range urls {
		churl <- data.Link{Url: url, Referrer: site.Root}
	}

	crawl(job)

	<-job.Finished
}

func printStatus(job *data.Job) {
	fmt.Printf("\rRoot: %v  Url queue: %d. Content queue: %d. Processed: %d. Errors: %d   \t", job.Site.Root, len(job.Urls), len(job.Content), job.Processed, job.Errors)
}

func crawl(job *data.Job) {
	for i := 0; i < runtime.NumCPU(); i++ {
		go getContent(job)
		go getLinks(job)
	}
}

func getStartUrlList(site data.Site) []string {
	list := []string{}
	list = append(list, site.Root)
	for _, i := range site.VirtualPaths {
		list = append(list, site.Root+i)
	}
	return list
}

func checkDone(job *data.Job) {
	allDone := job.Processed > 1 && len(job.Urls) == 0 && len(job.Content) == 0
	if allDone {
		job.Finished <- true
	}
}

func getContent(job *data.Job) {
	for {
		select {
		case url := <-job.Urls:
			ch, err := getUrlContents(job.Site, url)
			if err != nil {
				job.Errors++
			}
			job.Content <- ch

			printStatus(job)
			checkDone(job)
		}
	}
}

func getLinks(job *data.Job) {
	for {
		select {
		case cresp := <-job.Content:
			job.Processed++

			if cresp.Code == 200 {
				hrefs := parse(job.Site, cresp.Link.Url, cresp.Content, &job.Gathered)
				for _, href := range hrefs {
					job.Urls <- href
					job.Site.UrlsFile.WriteString(href.Url + "," + href.Referrer + "\n")
				}
			} else if cresp.Code >= 400 {
				job.Site.ErrorFile.WriteString(cresp.Link.Url + "," + cresp.Link.Referrer + "," + strconv.Itoa(cresp.Code) + "\n")
				job.Errors++
			}

			printStatus(job)
			checkDone(job)
		}
	}
}
