package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/guidj/manga-mirror/crawl"
	"github.com/guidj/manga-mirror/storage"
	"github.com/guidj/manga-mirror/utils"
)

// CrawlerQueue holds a sites and images channels
type CrawlerQueue struct {
	Sites  chan *url.URL
	Images chan *url.URL
	//	uri int <- lock issues. better to have a channel to receive a counter and increment or decrement it with a lock? atomic counters vs mutexes
}

// NewCrawlerQueue creates and returns an instance of a CrawlerQueue
func NewCrawlerQueue(size int) (cq *CrawlerQueue) {
	cq = new(CrawlerQueue)
	cq.Sites = make(chan *url.URL, size)
	cq.Images = make(chan *url.URL, size)
	return
}

// ManageQueues handles flow of data between `purgatory` and `waiting` CrawlerQueues
func ManageQueues(db *storage.KeyStore, purgatory, waiting *CrawlerQueue) {

	var uri *url.URL
	var img *url.URL

	for {

		select {
		case uri = <-purgatory.Sites:
			val, err := db.Get(uri.String())
			if err != nil {
				panic(err)
			}

			if val == "" {
				//log.Printf("Trying to add [%v] to crawl queue", uri.String())
				db.Save(uri.String(), "INQ")
				waiting.Sites <- uri
				log.Printf("Added site [%v] to waiting queue", uri.String())
				//c.urlIn += 1
			}

		case img = <-purgatory.Images:

			val, err := db.Get(img.String())
			if err != nil {
				panic(err)
			}

			if val == "" {
				db.Save(img.String(), "INQ")
				waiting.Images <- img
				log.Printf("Added image [%v] to waiting queue", img.String())
			}
		}
	}
}

// OperateNotifier tracks processed URL (for images and pages) and saves them to the index (storage)
func OperateNotifier(db *storage.KeyStore, processed *CrawlerQueue) {

	// TODO: counters?
	var uri *url.URL
	var img *url.URL

	for {
		select {
		case uri = <-processed.Sites:

			err := db.Save(uri.String(), "DONE")

			if err != nil {
				log.Println(err)
			}

			log.Printf("Marked site [%v] as processed", uri.String())

		case img = <-processed.Images:

			err := db.Save(img.String(), "DONE")

			if err != nil {
				log.Println(err)
			}

			log.Printf("Marked image [%v] as downloaded", img.String())
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

		validURL, err := utils.IsValidURL(*flDomain)

		if err != nil {
			log.Fatal(err)
		}

		if validURL == false {
			log.Fatalf("Invalid base URL [%v]", *flDomain)
		}
	}()

	if *flLogFile != "" {
		baseDir := path.Dir(*flLogFile)

		if _, err := os.Stat(baseDir); os.IsNotExist(err) {
			os.MkdirAll(baseDir, 0755)
		}

		f, err := os.OpenFile(*flLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Error opening file: %v", err)
		}
		log.SetOutput(f)
		defer f.Close()
	}

	log.Println("Initianting...")

	db := storage.NewKeyStore(*flDb)

	defer db.Close()

	db.Init(*flDb)

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
	purgatory, waiting, processed := NewCrawlerQueue(chSize), NewCrawlerQueue(chSize), NewCrawlerQueue(chSize)

	nCrawlers, nDownloaders, nHarversters := 6, 6, 3

	ticker := time.NewTicker(time.Millisecond * 5000)

	purgatory.Sites <- domain

	for i := 0; i < nCrawlers; i++ {
		go crawl.Crawl(i+1, waiting.Sites, processed.Sites, content)
	}

	for i := 0; i < nDownloaders; i++ {
		go crawl.Download(i+1, *flDir, waiting.Images, processed.Images)
	}

	go ManageQueues(db, purgatory, waiting)

	for i := 0; i < nHarversters; i++ {
		go crawl.Harvest(i+1, domain, filterRegex, content, purgatory.Sites, purgatory.Images)
	}

	go func() {

		for t := range ticker.C {
			// TODO: after n heartbeats of inactivity, close all channels to stop processing and exit

			log.Printf("Heartbeat @ %v", t)
		}
	}()

	OperateNotifier(db, processed)
}

// TODO: close channels, and end workers when exit/stop SIG is received
// TODO: count events? query db, or keep tabs on channels?
// TODO: waiting queue: just because an item is there, it doesn't mean it should be re-added
