package main

import (
	"flag"
	"log"
	"net/url"
	"strings"
	"time"
)

import "github.com/boltdb/bolt"
import "github.com/guidj/mangamirror/storage"
import "github.com/guidj/mangamirror/crawl"
import "github.com/guidj/mangamirror/utils"

type crawlCounter struct {
	//imageIn  int64
	//urlIn    int64
	//imageOut int64
	//urlOut   int64
}

type CrawlerQueue struct {
	Uri chan *url.URL
	Img chan *url.URL
	//	uri int64 <- lock issues. better to have a channel to receive a counter and increment or decrement it with a lock? atomic counters vs mutexes
}

//NewCrawlerQueue creates and returns an instance of a CrawlerQueue
func NewCrawlerQueue(size int64) (cq *CrawlerQueue) {
	cq = new(CrawlerQueue)
	cq.Uri = make(chan *url.URL, size)
	cq.Img = make(chan *url.URL, size)
	return
}

//ManageQueues handles flow of data between waiting and processed queues for URLs and Images
func ManageQueues(db *bolt.DB, newQueue, waitQueue *CrawlerQueue) {

	var uri *url.URL
	var img *url.URL

	for {

		select {
		case uri = <-newQueue.Uri:
			crawled, err := storage.Exists(db, uri.String())
			if err != nil {
				panic(err)
			}

			if crawled == false {
				//log.Printf("Trying to add [%v] to crawl queue", uri.String())
				waitQueue.Uri <- uri
				log.Printf("Added [%v] to crawl queue", uri.String())
				//c.urlIn += 1
			}

		case img = <-newQueue.Img:

			downloaded, err := storage.Exists(db, img.String())
			if err != nil {
				panic(err)
			}

			if downloaded == false {
				//log.Printf("Trying to add [%v] to download queue", img.String())
				waitQueue.Img <- img
				log.Printf("Added [%v] to download queue", img.String())
				//c.imageIn += 1
			}
		}
	}

	log.Println("Operator is exiting")
}

//OperateNotifier tracks processed URL (for images and pages) and saves them to the index (storage)
func OperateNotifier(db *bolt.DB, doneQueue *CrawlerQueue) {

	var uri *url.URL
	var img *url.URL
	for {
		select {
		case uri = <-doneQueue.Uri:

			err := storage.Save(db, uri.String())

			if err != nil {
				log.Println(err)
			}

			log.Printf("Added [%v] to crawled pages", uri.String())

			//c.urlOut += 1

		case img = <-doneQueue.Uri:

			err := storage.Save(db, img.String())

			if err != nil {
				log.Println(err)
			}

			log.Printf("Added [%v] to downloaded images", img.String())

			//c.imageOut += 1
		}
	}
}

func main() {

	var flDir string
	var flDb string
	var flDomain string

	func() {
		flag.StringVar(&flDir, "dir", "media", "Path to store downloaded media")
		flag.StringVar(&flDb, "db", ".mrdb", "Path for crawler sync index")
		flag.StringVar(&flDomain, "domain", "", "Web URL to crawl for media")

		flag.Parse()

		if len(flDomain) == 0 {
			log.Fatal("Domain is required")
		}

		validUrl, err := utils.IsValidUrl(flDomain)

		if err != nil {
			log.Fatal(err)
		}

		if validUrl == false {
			log.Fatalf("Invalid base URL [%v]", flDomain)
		}
	}()

	db, err := bolt.Open(flDb, 0600, nil)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	storage.Init(db)

	domain, err := url.Parse(flDomain)

	if err != nil {
		log.Fatal(err)
	}

	if domain.IsAbs() == false {
		log.Fatalf("Domain URL [%v] is not absolute", domain.String())
	}

	keywords := strings.Split("", ",")

	log.Printf("Domain: %v", domain.String())
	log.Printf("Keywords: %v", keywords)

	var chSize int64 = 1000
	content := make(chan string, chSize)
	newQueue, waitQueue, doneQueue := NewCrawlerQueue(chSize), NewCrawlerQueue(chSize), NewCrawlerQueue(chSize)

	nCrawlers, nDownloaders, nHarversters := 6, 6, 3
	nWorkers := nCrawlers * nDownloaders * nHarversters

	stop := make(chan int, nWorkers)

	//c := new(crawlCounter)
	ticker := time.NewTicker(time.Millisecond * 5000)

	newQueue.Uri <- domain

	for i := 0; i < 6; i++ {
		go crawl.Crawl(waitQueue.Uri, doneQueue.Uri, content, stop)
		go crawl.Download(i+1, flDir, waitQueue.Img, doneQueue.Img, stop)
	}

	go ManageQueues(db, newQueue, waitQueue)

	for i := 0; i < nHarversters; i++ {
		go crawl.Harvest(i+1, domain, keywords, content, newQueue.Uri, newQueue.Img, stop)
	}

	go func() {

		for t := range ticker.C {
			//log.Printf("Snapshot: Url(%v, %v), Image(%v, %v) @ %v",
			//	c.urlIn, c.urlOut, c.imageIn, c.imageOut, t)
			log.Printf("Heartbeat @ %v", t)
		}
	}()

	OperateNotifier(db, doneQueue)
}

//TODO: close channels, and end workers when exit/stop SIG is received
//TODO: count events? query db, or keep tabs on channels?
