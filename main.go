package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"time"
)

import "github.com/boltdb/bolt"
import "github.com/guidj/manga-mirror/storage"
import "github.com/guidj/manga-mirror/crawl"
import "github.com/guidj/manga-mirror/utils"

type crawlCounter struct {
	//imageIn  int
	//urlIn    int
	//imageOut int
	//urlOut   int
}

//CrawlerQueue holds a URI queue and an Image queue
type CrawlerQueue struct {
	URI chan *url.URL
	Img chan *url.URL
	//	uri int <- lock issues. better to have a channel to receive a counter and increment or decrement it with a lock? atomic counters vs mutexes
}

//NewCrawlerQueue creates and returns an instance of a CrawlerQueue
func NewCrawlerQueue(size int) (cq *CrawlerQueue) {
	cq = new(CrawlerQueue)
	cq.URI = make(chan *url.URL, size)
	cq.Img = make(chan *url.URL, size)
	return
}

//ManageQueues handles flow of data between waiting and processed queues for URLs and Images
func ManageQueues(db *bolt.DB, newQueue, waitQueue *CrawlerQueue) {

	var uri *url.URL
	var img *url.URL

	for {

		select {
		case uri = <-newQueue.URI:
			val, err := storage.Get(db, uri.String())
			if err != nil {
				panic(err)
			}

			if val == "" {
				//log.Printf("Trying to add [%v] to crawl queue", uri.String())
				storage.Save(db, uri.String(), "INQ")
				waitQueue.URI <- uri
				log.Printf("Added [%v] to crawl queue", uri.String())
				//c.urlIn += 1
			}
			//else if val == "INQ" {
			//do nothing
			//}

		case img = <-newQueue.Img:

			val, err := storage.Get(db, img.String())
			if err != nil {
				panic(err)
			}

			if val == "" {
				//log.Printf("Trying to add [%v] to download queue", img.String())
				storage.Save(db, img.String(), "INQ")
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
		case uri = <-doneQueue.URI:

			err := storage.Save(db, uri.String(), "DONE")

			if err != nil {
				log.Println(err)
			}

			log.Printf("Added [%v] to crawled pages", uri.String())

			//c.urlOut += 1

		case img = <-doneQueue.Img:

			err := storage.Save(db, img.String(), "DONE")

			if err != nil {
				log.Println(err)
			}

			log.Printf("Added [%v] to downloaded images", img.String())

			//c.imageOut += 1
		}
	}
}

func main() {

	var flDir = flag.String("dir", "_media", "Path to store downlaoded media")
	var flDb = flag.String("db", "_mrdb", "Path for crawler sync index")
	var flDomain = flag.String("url", "", "Web URL to crawl for media")
	var flFilterRegex = flag.String("filter", "", "Regex pattern to filter URIs, e.g. 'mangareader.net|naruto'")
	var flLogFile = flag.String("log", "", "Log file. Defaults to outputting to STDOUT")

	func() {

		flag.Parse()

		if *flDomain == "" {
			log.Fatal("Domain is required")
		}

		validURL, err := utils.IsValidUrl(*flDomain)

		if err != nil {
			log.Fatal(err)
		}

		if validURL == false {
			log.Fatalf("Invalid base URL [%v]", *flDomain)
		}
	}()

	if *flLogFile != "" {
		f, err := os.OpenFile(*flLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		log.SetOutput(f)
		defer f.Close()
	}

	log.Println("Initianting...")

	db, err := bolt.Open(*flDb, 0600, nil)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	storage.Init(db)

	domain, err := url.Parse(*flDomain)

	if err != nil {
		log.Fatal(err)
	}

	if domain.IsAbs() == false {
		log.Fatalf("Domain URL [%v] is not absolute", domain.String())
	}

	filterRegex := *flFilterRegex

	log.Printf("Domain: %v", domain.String())
	log.Printf("Filter (regex): %v", filterRegex)

	chSize := 1000
	content := make(chan string, chSize)
	newQueue, waitQueue, doneQueue := NewCrawlerQueue(chSize), NewCrawlerQueue(chSize), NewCrawlerQueue(chSize)

	nCrawlers, nDownloaders, nHarversters := 6, 6, 3
	nWorkers := nCrawlers * nDownloaders * nHarversters

	stop := make(chan int, nWorkers)

	ticker := time.NewTicker(time.Millisecond * 5000)

	newQueue.URI <- domain

	for i := 0; i < nCrawlers; i++ {
		go crawl.Crawl(i+1, waitQueue.URI, doneQueue.URI, content, stop)
	}

	for i := 0; i < nDownloaders; i++ {
		go crawl.Download(i+1, *flDir, waitQueue.Img, doneQueue.Img, stop)
	}

	go ManageQueues(db, newQueue, waitQueue)

	for i := 0; i < nHarversters; i++ {
		go crawl.Harvest(i+1, domain, filterRegex, content, newQueue.URI, newQueue.Img, stop)
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
//TODO: waiting queue: just because an item is there, it doesn't mean it should be re-added
