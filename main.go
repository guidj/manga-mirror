package main

import (
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

//TODO: package naming
type crawlCounter struct {
	queuedImages     int64
	queuedUrls       int64
	downloadedImages int64
	crawledPages     int64
}

func OperateFlow(db *bolt.DB, parsedPages, parsedImages <-chan *url.URL, pagesToCrawl, imagesToDownload chan<- *url.URL, c *crawlCounter) {

	for {

		select {
		case parsedPage := <-parsedPages:

			crawled, err := IsPageCrawled(db, parsedPage)
			if err != nil {
				panic(err)
			}

			if crawled == false {
				log.Printf("Trying to add [%v] to crawl queue", parsedPage.String())
				pagesToCrawl <- parsedPage
				log.Printf("Added [%v] to crawl queue", parsedPage.String())
				c.queuedUrls = c.queuedUrls + 1
			}

		case parsedImage := <-parsedImages:

			downloaded, err := IsImageCrawled(db, parsedImage)
			if err != nil {
				panic(err)
			}

			if downloaded == false {
				log.Printf("Trying to add [%v] to download queue", parsedImage.String())
				imagesToDownload <- parsedImage
				log.Printf("Added [%v] to download queue", parsedImage.String())
				c.queuedImages = c.queuedImages + 1
			}
		}
	}

	log.Println("Operator is exiting")
}

func OperateNotifier(db *bolt.DB, crawledPages, downloadedImages <-chan *url.URL, c *crawlCounter) {

	for {
		select {
		case crawledPage := <-crawledPages:

			log.Printf("Trying to add [%v] to crawled pages", crawledPage.String())
			err := AddPage(db, crawledPage)

			if err != nil {
				log.Println(err)
			}

			log.Printf("Added [%v] to crawled pages", crawledPage.String())

			c.crawledPages = c.crawledPages + 1

		case downloadedImage := <-downloadedImages:

			log.Printf("Trying to add [%v] to downloaded images", downloadedImage.String())
			err := AddImage(db, downloadedImage)

			if err != nil {
				log.Println(err)
			}

			log.Printf("Added [%v] to downloaded images", downloadedImage.String())

			c.downloadedImages = c.downloadedImages + 1
		}
	}
}

func main() {

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

	c := new(crawlCounter)
	ticker := time.NewTicker(time.Millisecond * 500)

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

	for i := 0; i < 6; i++ {
		go Crawl(pagesToCrawl, crawledPages, content)
		go Collect(i+1, imagesToDownload, downloadedImages, stop)
		go OperateFlow(db, parsedPages, parsedImages, pagesToCrawl, imagesToDownload, c)
	}

	for i := 0; i < 2; i++ {
		go Harvest(i+1, baseUrl, keywords, content, parsedPages, parsedImages, stop)
	}

	go func() {

		for t := range ticker.C {
			log.Printf("Snapshot: Url(%v, %v), Image(%v, %v) @ %v",
				c.queuedUrls, c.crawledPages, c.queuedImages, c.downloadedImages,
				t)
		}
	}()

	OperateNotifier(db, crawledPages, downloadedImages, c)
}
