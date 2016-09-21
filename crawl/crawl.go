package crawl

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

func ParseElementValues(html, tag, element string) []string {
	stf := fmt.Sprint(`<`, tag, `[^>]+`, element)
	pattern := fmt.Sprintf(`%v\s*=\s*'(.*?)'|%v\s*=\s*"(.*?)"`, stf, stf)
	re := regexp.MustCompile(pattern)
	elements := re.FindAllStringSubmatch(html, -1)

	values := make([]string, len(elements))

	for i, v := range elements {
		values[i] = strings.TrimSpace(v[len(v)-1])
	}

	return values
}

//TODO: keywords filtering
func MakeURIParser(tag, element string, domain *url.URL) func(html string) []*url.URL {
	return func(html string) []*url.URL {

		paths := ParseElementValues(html, tag, element)
		urls := make([]*url.URL, len(paths))

		var uri *url.URL
		var err error

		for i, path := range paths {
			uri, err = url.Parse(path)

			if err != nil {
				log.Printf("Invalid url path: %v", path)
				continue
			}

			if uri.IsAbs() {
				urls[i] = uri
			} else {
				urls[i] = domain.ResolveReference(uri)
			}
		}

		return urls
	}
}

//TODO: is it better to pass data in channels by copying (less memory sharing)? (applies to domain as well)

func ParseContent(u *url.URL) (string, error) {

	resp, err := http.Get(u.String())

	if err != nil {
		log.Println(err)

		return string(""), err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	content := string(body)

	return content, err
}

//Harvest extracts URI of images and other URLs form an html string
func Harvest(id int, domain *url.URL, keywords []string, content <-chan string, urlQueue, imageQueue chan<- *url.URL, stop <-chan int) {

	var mURLs []*url.URL
	var mImages []*url.URL

	var mParseURLs = MakeURIParser("a", "href", domain)
	var mParseImages = MakeURIParser("img", "src", domain)

	var c string

	for {
		log.Printf("Harvester %v entering loop", id)

		select {
		case c = <-content:

			//newURLs := FindUrls(c, domain, keywords)
			//newIMGs := FindImages(c, domain, keywords)
			//TODO: run this concurrently
			mURLs = mParseURLs(c)
			mImages = mParseImages(c)

			log.Printf("Parsed %v new URL(s) and %v image(s)", len(mURLs), len(mImages))

			for _, u := range mURLs {
				//log.Printf("Trying to add [%v] to parsed pages", u.String())
				urlQueue <- u
				log.Printf("Added [%v] to parsed pages", u.String())
			}

			for _, i := range mImages {
				//log.Printf("Trying to add [%v] to parsed images", i.String())
				imageQueue <- i
				log.Printf("Adding [%v] to parsed images", i.String())
			}
		case <-stop:
			log.Printf("Harvester recieved stop call. Exiting...")
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	//TODO: call this when a signal is received to stop the loop
	log.Printf("Havester %v is exiting", id)
}

//Crawl parses URLs from a input queue, places the content in a content queue, and places the URL on an ouput queue
func Crawl(inQueue <-chan *url.URL, outQueue chan<- *url.URL, content chan<- string, stop <-chan int) {

	var u *url.URL
	for {
		select {
		case u = <-inQueue:
			c, err := ParseContent(u)

			log.Printf("Parsed content for [%v]", u.String())

			if err != nil {
				log.Println(err)
			} else {

				//log.Printf("Trying to add html from [%v] to content", u.String())
				content <- c
				log.Printf("Added html from [%v] to content", u.String())

				//log.Printf("Trying to add [%v] to crawled pages", u.String())
				outQueue <- u
				log.Printf("Added [%v] to crawled pages", u.String())
			}

			//TODO: deal with failed crawls
		case <-stop:
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("Crawler is exiting")
}

//TODO: download resource (rename)
//TODO: check if channel is closed, and stop
//Download downloads resources from URIs URI to a given path
func Download(id int, dir string, inQueue <-chan *url.URL, outQueue chan<- *url.URL, stop <-chan int) {

	log.Printf("Collector %v entering loop", id)

	var i *url.URL

	for {
		select {
		case i = <-inQueue:
			//TODO: get data dir as a param from user running program
			p := path.Join(dir, i.Path)

			if _, err := os.Stat(path.Dir(p)); os.IsNotExist(err) {
				os.MkdirAll(path.Dir(p), 0755)
			}

			resp, err := http.Get(i.String())

			if err != nil {
				log.Println(err)
			} else {

				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)

				ioutil.WriteFile(p, body, 0755)

				log.Printf("Downloaded %v, to %v", i.String(), p)

				if err != nil {
					log.Println(err)
				}

				//TODO: add to failure list/queue
				outQueue <- i
			}

		case <-stop:
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	//TODO: call this when a signal is received to stop the loop
	log.Printf("Collector %v exiting", id)
}

//TODO: robots.txt
