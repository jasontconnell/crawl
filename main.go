package main

import (
    "fmt"
    "runtime"
    "time"
    "sync"
    "encoding/json"
    "os"
    "net/http"
    "net/url"
    "io/ioutil"
    "regexp"
    "strings"
)

var hrefRegex *regexp.Regexp = regexp.MustCompile(`(href|src)="(.*?)"`)
var gatheredUrls map[string]string = make(map[string]string)

type Headers map[string]string

type Site struct {
    Root string                 `json:"root"`
    VirtualPaths []string       `json:"virtualPaths"`
    Headers Headers             `json:"headers"`
    ErrorFileName string            `json:"errorFile"`
    ErrorFile *os.File
}

type Link struct {
    Referrer string
    Url string
}

type ContentResponse struct {
    Link Link
    Code int
    Content string
}

func Parse(site Site, referrer, content string) (urls chan Link) {
    var wg sync.WaitGroup
    wg.Add(1)
    urls = make(chan Link)

    go func(){
        defer close(urls)
        defer wg.Done()

        matches := hrefRegex.FindAllStringSubmatch(content, -1)
        for _,m := range matches {
            href := strings.Trim(m[2], " ")
            add := (href != "" && strings.HasPrefix(href, "/") && !strings.HasPrefix(href, "//") && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "mailto:") && !strings.HasPrefix(href, "javascript")) || strings.HasPrefix(href, site.Root)
            if strings.HasPrefix(href, "/") {
                href = site.Root + href
            }

            _,contains := gatheredUrls[href]
            _,contains2 := gatheredUrls[href + "/"]
            _,contains3 := gatheredUrls[strings.TrimSuffix(href, "/")]
            contains = contains || contains2 || contains3

            if !contains {
                gatheredUrls[href] = href
            }

            if add && !contains {
                urls <- Link{ Referrer: referrer, Url: href }
            }
        }
    }()

    return
}

func init(){
    runtime.GOMAXPROCS(runtime.NumCPU())
}

func main(){
    startTime := time.Now()

    if file, err := os.OpenFile("entrypoints.json", os.O_RDONLY, os.ModePerm); err == nil {
        enc := json.NewDecoder(file)

        var site Site
        jserr := enc.Decode(&site)

        site.ErrorFile,_ = os.Create(site.ErrorFileName)
        site.ErrorFile.Chmod(os.ModeAppend)
        site.ErrorFile.WriteString("Url, Referrer, Code\n")


        if jserr == nil {
            urls := getStartUrlList(site)
            churl := make(chan Link, 15000)
            chcresp := make(chan ContentResponse, 300)
            chproc := make(chan ContentResponse, 15000)
            cherrs := make(chan ContentResponse, 300)
            errCount := 0
            finished := make(chan bool)
               
            go process(site, churl, chcresp, chproc, cherrs, &errCount, finished)
         
            for _,url := range urls {
                gatheredUrls[url] = url
                churl <- Link{ Referrer: site.Root, Url: url}
            }

            <-finished
        } else {
            fmt.Println(jserr)
        }
    }
    
    fmt.Println("done. time", time.Since(startTime))
}

func getStartUrlList(site Site) (list []string) {
    list = []string{}
    list = append(list, site.Root)
    for _,i := range site.VirtualPaths {
        list = append(list, site.Root + i)
    }
    return
}

func process(site Site, urls chan Link, content, processed, errors chan ContentResponse, errCount *int, finished chan bool){
    for {
        select {
        case url := <-urls:
            
            fmt.Printf("\rUrl queue: %d. Content queue: %d. Processed: %d. Errors: %d\t", len(urls), len(content), len(processed), *errCount)
            ch := getUrlContents(site, url)
            content <- <-ch

        case cresp := <-content:
            processed <- cresp
            fmt.Printf("\rUrl queue: %d. Content queue: %d. Processed: %d. Errors: %d\t", len(urls), len(content), len(processed), *errCount)

            if cresp.Code == 200 {
                hrefs := Parse(site, cresp.Link.Url, cresp.Content)
                for href := range hrefs {
                    urls <- href
                }
            } else {
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

func getUrlContents(site Site, link Link) (chresp chan ContentResponse){
    var wg sync.WaitGroup
    wg.Add(1)
    chresp = make(chan ContentResponse)

    go func(){
        defer close(chresp)
        defer wg.Done()

        uri,parseErr := url.Parse(link.Url)
        if parseErr != nil {
            panic(parseErr)
        }

        headers := make(map[string][]string)
        for k,v := range site.Headers {
            headers[k] = []string{ v }
        }

        req := &http.Request{ Method: "GET", URL: uri, Header: headers }
        client := &http.Client{
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                if strings.Contains(site.Root, req.URL.Host){
                    return nil
                }
                return http.ErrUseLastResponse
            },
        }

        resp,err := client.Do(req)
        defer resp.Body.Close()

        if err == nil {
            if contents, err := ioutil.ReadAll(resp.Body); err == nil {
                text := string(contents)
                chresp <- ContentResponse { Link: link, Code: resp.StatusCode, Content: text }
            } else {
                fmt.Println(err)
            }
        } else {
            fmt.Println(err)
        }
    }()
    
    return
}