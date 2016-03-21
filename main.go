package main

import (
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/boltdb/bolt"
)

var count int64 // *Error* non-declaration statement outside function body

func increment() error {
	count = count + 1
	return nil
}

func OperateFlow(db *bolt.DB, parsedPages, parsedImages, crawledPages, downloadedImages <-chan *url.URL, pagesToCrawl, imagesToDownload chan<- *url.URL) {

	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Crawl.Pages"))
		if err != nil {
			panic(err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("Crawl.Images"))
		if err != nil {
			panic(err)
		}

		return nil
	})

	for {

		select {
		case parsedPage := <-parsedPages:

			go func() {
				crawled, err := IsPageCrawled(db, parsedPage)
				if err != nil {
					panic(err)
				}

				if crawled == false {
					pagesToCrawl <- parsedPage
				}
			}()

		case parsedImage := <-parsedImages:

			go func() {
				crawled, err := IsImageCrawled(db, parsedImage)
				if err != nil {
					panic(err)
				}

				if crawled == false {
					imagesToDownload <- parsedImage
				}
			}()

		case crawledPage := <-crawledPages:
			AddPage(db, crawledPage)
		case downloadedImage := <-downloadedImages:
			AddImage(db, downloadedImage)
		}
	}

}

func main() {

	db, err := bolt.Open("idx/crawler.db", 0600, nil)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	params := os.Args[1:]

	if len(params) < 2 {
		log.Fatal("Requires base URL as a parameter, and keywords")
	}

	validUrl, err := IsValidUrl(params[0])
	keywords := strings.Split(params[1], ",")

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

	log.Printf("Base URL: %v", baseUrl.String())
	log.Printf("Keywords: %v", keywords)

	content := make(chan string, 1000)
	parsedPages := make(chan *url.URL, 1000)
	parsedImages := make(chan *url.URL, 1000)
	pagesToCrawl := make(chan *url.URL, 1000)
	imagesToDownload := make(chan *url.URL, 1000)
	crawledPages := make(chan *url.URL, 1000)
	downloadedImages := make(chan *url.URL, 1000)
	stop := make(chan int, 1)

	pagesToCrawl <- baseUrl

	for i := 0; i < 4; i++ {
		go Crawl(pagesToCrawl, crawledPages, content)
		go Collect(i, imagesToDownload, downloadedImages, stop)
		go Harvest(i, baseUrl, keywords, content, parsedPages, parsedImages, stop)
	}

	OperateFlow(db, parsedPages, parsedImages, crawledPages, downloadedImages, pagesToCrawl, imagesToDownload)

}

//TODO: check downloaded file permission
//TODO: packaging
