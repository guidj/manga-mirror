// crawl
package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
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

func ParseBody(u *url.URL) (string, error) {

	resp, err := http.Get(u.String())

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	html := string(body)

	return html, err
}

func main() {

	params := os.Args[1:]

	if len(params) < 1 {
		log.Fatal("Requires base URL as a parameter")
	}

	validUrl, err := IsValidUrl(params[0])

	if err != nil {
		log.Fatal(err)
	}

	if validUrl == false {
		log.Fatalf("Invalid base URL [%v]", params[0])
	}

	baseUrl, err := url.Parse(params[0])

	if err != nil {
		log.Fatal(err)
	}

	if baseUrl.IsAbs() == false {
		log.Fatalf("Base URL [%v] is not absolute", baseUrl.String())
	}

	html, err := ParseBody(baseUrl)

	if err != nil {
		panic(err)
	}

	urls := FindUrls(html, baseUrl, []string{})
	imageUrls := FindImages(html, baseUrl, []string{})

	log.Printf("Found %v URL(s) and %v Image(s)", len(urls), len(imageUrls))

	for _, u := range urls {
		log.Printf("URL: %v", u)
	}

	for _, u := range imageUrls {
		log.Printf("Image: %v", u)
	}
}

//TODO: robots.txt
//TODO: download images
//TODO: go routines for concurrent processing with channels
