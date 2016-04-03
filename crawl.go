// crawl
package main

import (
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

func IsValidUrl(url string) (bool, error) {
	pattern := "(@)?(href=')?(HREF=')?(HREF=\")?(href=\")?(http[s]?://)?[a-zA-Z_0-9\\-]+(\\.\\w[a-zA-Z_0-9\\-]+)+(/[#&\\n\\-=?\\+\\%/\\.\\w]+)?"

	valid, err := regexp.MatchString(pattern, url)

	return valid, err
}

func FindUrls(html string, baseUrl *url.URL, keywords []string) []*url.URL {

	index := strings.Index(html, "href")
	urls := make(map[string]*url.URL)

	for index != -1 {
		openingTagIndex := index + 5
		openingTag := string(html[openingTagIndex])
		closingTagIndex := strings.Index(html[openingTagIndex+1:], openingTag)

		u, err := url.Parse(html[openingTagIndex+1 : openingTagIndex+closingTagIndex+1])

		if err != nil {

			log.Printf("Invalid url: %v", u)
			html = html[closingTagIndex+1:]
			index = strings.Index(html, "<img")

			continue

		}

		if u.IsAbs() == false {
			u = baseUrl.ResolveReference(u)
		}

		add := true

		for _, keyword := range keywords {
			if strings.Contains(u.String(), keyword) == false {
				add = false
				break
			}
		}

		if add {
			_, ok := urls[u.String()]

			if ok == false {
				urls[u.String()] = u
			}
		}

		html = html[closingTagIndex+1:]
		index = strings.Index(html, "href")
	}

	list, i := make([]*url.URL, len(urls)), 0

	for _, v := range urls {
		list[i] = v
		i++
	}

	return list
}

func FindImages(html string, baseUrl *url.URL, keywords []string) []*url.URL {

	index := strings.Index(html, "<img")
	urls := make(map[string]*url.URL)

	for index != -1 {

		html = html[index+1:]

		sourceIndex := strings.Index(html, "src")

		openingTagIndex := sourceIndex + 4
		openingTag := string(html[openingTagIndex])
		closingTagIndex := strings.Index(html[openingTagIndex+1:], openingTag)

		u, err := url.Parse(html[openingTagIndex+1 : openingTagIndex+closingTagIndex+1])

		if err != nil {

			log.Printf("Invalid url: %v", u)
			html = html[closingTagIndex+1:]
			index = strings.Index(html, "<img")

			continue
		}

		if u.IsAbs() == false {
			u = baseUrl.ResolveReference(u)
		}

		add := true

		for _, keyword := range keywords {
			if strings.Contains(u.String(), keyword) == false {
				add = false
				break
			}
		}

		if add {
			_, ok := urls[u.String()]

			if ok == false {
				urls[u.String()] = u
			}
		}

		html = html[closingTagIndex+1:]
		index = strings.Index(html, "<img")
	}

	list, i := make([]*url.URL, len(urls)), 0

	for _, v := range urls {
		list[i] = v
		i++
	}

	return list
}

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

func Harvest(id int, baseUrl *url.URL, keywords []string, content <-chan string, parsedPages chan<- *url.URL, parsedImages chan<- *url.URL, stop <-chan int) {

	for {
		log.Printf("Harvester %v entering loop", id)

		select {
		case c := <-content:

			newURLs := FindUrls(c, baseUrl, keywords)
			newIMGs := FindImages(c, baseUrl, keywords)

			log.Printf("Parsed %v new URL(s) and %v image(s)", len(newURLs), len(newIMGs))

			for _, u := range newURLs {
				log.Printf("Trying to add [%v] to parsed pages", u.String())
				parsedPages <- u
				log.Printf("Added [%v] to parsed pages", u.String())
			}

			for _, i := range newIMGs {
				log.Printf("Trying to add [%v] to parsed images", i.String())
				parsedImages <- i
				log.Printf("Adding [%v] to parsed images", i.String())
			}
		case <-stop:
			log.Printf("Harvester recieved stop call. Exiting...")
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("Havester %v is exiting", id)
}

func Crawl(pagesToCrawl <-chan *url.URL, crawledPages chan<- *url.URL, content chan<- string) {

	for {
		select {
		case u := <-pagesToCrawl:
			c, err := ParseContent(u)

			log.Printf("Parsed content for [%v]", u.String())

			if err != nil {
				log.Println(err)
			} else {

				log.Printf("Trying to add html from [%v] to content", u.String())
				content <- c
				log.Printf("Added html from [%v] to content", u.String())

				log.Printf("Trying to add [%v] to crawled pages", u.String())
				crawledPages <- u
				log.Printf("Added [%v] to crawled pages", u.String())
			}

			//TODO: deal with failed crawls
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func Collect(id int, imagesToDownload <-chan *url.URL, downloadedImages chan<- *url.URL, stop <-chan int) {

	for {
		log.Printf("Collector %v entering loop", id)

		select {
		case i := <-imagesToDownload:

			p := path.Join("repo/", i.Path)

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
				downloadedImages <- i
			}

		case <-stop:
			return
		}

		time.Sleep(500 * time.Millisecond)
	}

	log.Printf("Collector %v exiting", id)
}

//TODO: robots.txt
