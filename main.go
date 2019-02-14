package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var hrefRegex *regexp.Regexp = regexp.MustCompile(`(href|src)="(.*?)"`)

type Headers map[string]string

type Site struct {
	Root            string   `json:"root"`
	VirtualPaths    []string `json:"virtualPaths"`
	Headers         Headers  `json:"headers"`
	ErrorFileName   string   `json:"errorFile"`
	ErrorFile       *os.File
	AllUrlsFileName string `json:"allUrlsFile"`
	AllUrlsFile     *os.File
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

func Parse(site Site, referrer, content string, gatheredUrls sync.Map) (urls chan Link) {
	var wg sync.WaitGroup
	wg.Add(1)
	urls = make(chan Link)

	go func() {
		defer close(urls)
		defer wg.Done()

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
			}

			if add && !contains {
				urls <- Link{Referrer: referrer, Url: href}
			}
		}
	}()

	return
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	startTime := time.Now()

	file, err := os.Open("entrypoints.json")
	if err != nil {
		log.Fatal(err)
	}

	enc := json.NewDecoder(file)

	var site Site
	jserr := enc.Decode(&site)
	if jserr != nil {
		log.Fatal(jserr)
	}

	site.ErrorFile, _ = os.Create(site.ErrorFileName)
	site.ErrorFile.Chmod(os.ModeAppend)
	site.ErrorFile.WriteString("Url, Referrer, Code\n")

	site.AllUrlsFile, _ = os.Create(site.AllUrlsFileName)
	site.AllUrlsFile.Chmod(os.ModeAppend)
	site.AllUrlsFile.WriteString("Url, Referrer\n")
	site.AllUrlsFile.WriteString(site.Root + ",[Root]\n")

	gatheredUrls := sync.Map{}
	urls := getStartUrlList(site)
	churl := make(chan Link, 1500000)
	chcresp := make(chan ContentResponse, 300)
	chproc := make(chan ContentResponse, 1500000)
	cherrs := make(chan ContentResponse, 300)
	errCount := 0
	finished := make(chan bool)

	for i := 0; i < 8; i++ {
		go process(site, churl, chcresp, chproc, cherrs, &errCount, gatheredUrls, finished)
	}

	for _, url := range urls {
		// gatheredUrls[url] = url
		churl <- Link{Referrer: site.Root, Url: url}
	}

	<-finished

	fmt.Println("\n\ndone. time", time.Since(startTime))
}

func getStartUrlList(site Site) (list []string) {
	list = []string{}
	list = append(list, site.Root)
	for _, i := range site.VirtualPaths {
		list = append(list, site.Root+i)
	}
	return
}

func process(site Site, urls chan Link, content, processed, errors chan ContentResponse, errCount *int, gatheredUrls sync.Map, finished chan bool) {
	for {
		select {
		case url := <-urls:

			fmt.Printf("\rRoot: %v  Url queue: %d. Content queue: %d. Processed: %d. Errors: %d\t", site.Root, len(urls), len(content), len(processed), *errCount)
			ch := getUrlContents(site, url)
			content <- <-ch

		case cresp := <-content:
			processed <- cresp
			fmt.Printf("\rRoot: %v  Url queue: %d. Content queue: %d. Processed: %d. Errors: %d\t", site.Root, len(urls), len(content), len(processed), *errCount)

			if cresp.Code == 200 {
				hrefs := Parse(site, cresp.Link.Url, cresp.Content, gatheredUrls)
				for href := range hrefs {
					urls <- href
					// unique urls.
					site.AllUrlsFile.WriteString(href.Url + "," + href.Referrer + "\n")
				}
			} else if cresp.Code >= 400 {
				errors <- cresp
			}

			allDone := len(processed) > 1 && len(urls) == 0 && len(content) == 0
			if allDone {
				finished <- true
			}
		case err := <-errors:
			(*errCount)++
			site.ErrorFile.WriteString(fmt.Sprintf("%v, %v, %v\n", err.Link.Url, err.Link.Referrer, err.Code))
		}
	}
}

func getUrlContents(site Site, link Link) (chresp chan ContentResponse) {
	var wg sync.WaitGroup
	wg.Add(1)
	chresp = make(chan ContentResponse)

	go func() {
		defer close(chresp)
		defer wg.Done()

		uri, parseErr := url.Parse(link.Url)
		if parseErr != nil {
			log.Fatal(parseErr)
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
			log.Fatal(err)
		}

		defer resp.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			log.Fatal(err)
		}

		text := string(contents)
		chresp <- ContentResponse{Link: link, Code: resp.StatusCode, Content: text}
	}()

	return
}
