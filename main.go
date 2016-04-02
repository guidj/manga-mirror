package main

import (
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

type crawlCounter struct {
	queuedImages     int64
	queuedUrls       int64
	downloadedImages int64
	crawledPages     int64
}

func OperateFlow(db *bolt.DB, parsedPages, parsedImages, crawledPages, downloadedImages <-chan *url.URL, pagesToCrawl, imagesToDownload chan<- *url.URL, ticker <-chan time.Time) {

	c := crawlCounter{}

	for {

		select {
		case parsedPage := <-parsedPages:

			crawled, err := IsPageCrawled(db, parsedPage)
			if err != nil {
				panic(err)
			}

			if crawled == false {
				pagesToCrawl <- parsedPage
				c.queuedUrls = c.queuedUrls + 1
			}

		case parsedImage := <-parsedImages:

			crawled, err := IsImageCrawled(db, parsedImage)
			if err != nil {
				panic(err)
			}

			if crawled == false {
				imagesToDownload <- parsedImage
				c.queuedImages = c.queuedImages + 1
			}

		case crawledPage := <-crawledPages:
			AddPage(db, crawledPage)
			c.crawledPages = c.crawledPages + 1

		case downloadedImage := <-downloadedImages:
			AddImage(db, downloadedImage)
			c.downloadedImages = c.downloadedImages + 1

		case t := <-ticker:
			log.Printf("Snapshot: Url(%v, %v), Image(%v, %v) @ %v",
				c.queuedUrls, c.crawledPages, c.queuedImages, c.downloadedImages,
				t)
		}
	}

}

func main() {

	// param to delete prev crawling data

	db, err := bolt.Open("spider.db", 0600, nil)

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

	ticker := time.NewTicker(time.Millisecond * 500)

	for i := 0; i < 6; i++ {
		go Crawl(pagesToCrawl, crawledPages, content)
		go Collect(i, imagesToDownload, downloadedImages, stop)
	}

	for i := 0; i < 2; i++ {
		go Harvest(i, baseUrl, keywords, content, parsedPages, parsedImages, stop)
	}

	OperateFlow(db, parsedPages, parsedImages, crawledPages, downloadedImages, pagesToCrawl, imagesToDownload, ticker.C)
}

//TODO: check downloaded file permission
//TODO: packaging
