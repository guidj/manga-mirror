package crawl

import (
	"fmt"
	"log"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

import "github.com/guidj/manga-mirror/utils"

// ParseHTMLElementValues parses a specified html `element` with a specified `tag`.
func ParseHTMLElementValues(html, tag, element string) []string {
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

// MakeURIParser returns a function that takes a string with `html` and returns a list of URIs that match a given regex pattern.
// If the parsed URI is a relative URL, the `domain` URL is used to resolve it to an absolute path.
func MakeURIParser(tag, element string, domain *url.URL, filterPattern string) func(html string) []*url.URL {

	matchFilter := utils.MakeRegexMatcher(filterPattern)

	return func(html string) []*url.URL {

		paths := ParseHTMLElementValues(html, tag, element)
		urls := []*url.URL{}

		var uri *url.URL
		var err error

		for _, path := range paths {
			uri, err = url.Parse(path)

			if err != nil {
				log.Printf("Invalid url path: %v", path)
				continue
			}

			if !uri.IsAbs() {
				uri = domain.ResolveReference(uri)
			}

			if matchFilter(uri.String()) {
				urls = append(urls, uri)
			}
		}

		return urls
	}
}

//TODO: is it better to pass data in channels by copying (less memory sharing)? (applies to domain as well)

// Harvest extracts URIs from `anchor` (a) and `image` (img) tags in an HTML string, from htlm strings from a `content` channel
// `domain` is used to resolve the full URL of relative URLs
func Harvest(id int, domain *url.URL, filterPattern string, content <-chan string, sites, images chan<- *url.URL) {

	var siteURLs []*url.URL
	var imageURLs []*url.URL

	var fnParseSiteURLs = MakeURIParser("a", "href", domain, filterPattern)
	var fnParseImageURLs = MakeURIParser("img", "src", domain, filterPattern)

	var c string
	var open bool

	log.Printf("Harvest Worker-[%v] entering loop", id)

	for {
		select {
		case c, open = <-content:

			if open {
				siteURLs = fnParseSiteURLs(c)
				imageURLs = fnParseImageURLs(c)

				log.Printf("Harvest Worker-[%v] parsed %v new site and %v image URL(s)", id, len(siteURLs), len(imageURLs))

				for _, siteURL := range siteURLs {
					//log.Printf("Trying to add [%v] to parsed pages", u.String())
					sites <- siteURL
					log.Printf("Harvest Worker-[%v] added [%v] to parsed pages", id, siteURL.String())
				}

				for _, imageURL := range imageURLs {
					//log.Printf("Trying to add [%v] to parsed images", i.String())
					images <- imageURL
					log.Printf("Harvest Worker-[%v] added [%v] to parsed images", id, imageURL.String())
				}
			} else {
				log.Printf("Havest Worker-[%v] is exiting", id)
				break
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// Crawl parses URLs from a `waiting` channel, places the content in a `content` channel, and places the URL on an `processed` channel.
func Crawl(id int, userAgent string, waiting <-chan *url.URL, processed chan<- *url.URL, content chan<- string) {

	var u *url.URL
	var open bool
	var httpClient = utils.NewHttpClient(userAgent)

	for {
		select {
		case u, open = <-waiting:

			if open {

				c, err := httpClient.RetrieveContent(u.String())

				log.Printf("Crawl Worker-[%v] parsed content for [%v]", id, u.String())

				//TODO: deal with failed crawls, e.g. log with special value in key-store
				if err != nil {
					log.Println(err)
				} else {

					content <- c
					log.Printf("Crawl Worker-[%v] added html from [%v] to content", id, u.String())

					processed <- u
					log.Printf("Crawl Worker-[%v] added [%v] to crawled pages", id, u.String())
				}
			} else {
				log.Printf("Crawl Worker-[%v] is exiting", id)
				break
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

// Download downloads resources from URIs in a `waiting` channel and URI to a given `dir` and puts the URI in a `processed` channel
func Download(id int, userAgent string, dir string, wainting <-chan *url.URL, processed chan<- *url.URL) {

	log.Printf("Download Worker-[%v] entering loop", id)

	var i *url.URL
	var open bool
	var httpClient = utils.NewHttpClient(userAgent)

	for {
		select {
		case i, open = <-wainting:

			if open {

				//TODO: get data dir as a param from user running program
				filePath := path.Join(dir, i.Path)

				err := httpClient.Download(i.String(), filePath)

				if err != nil {
					log.Println(err)
				}

				//TODO: add to failure list/queue
				processed <- i

			} else {

				log.Printf("Download Worker-[%v] is exiting", id)
				break
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}

// TODO: robots.txt
