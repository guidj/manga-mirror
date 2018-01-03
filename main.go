package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/temoto/robotstxt"

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
func ManageQueues(db *storage.KeyStore, userAgent string, domain *url.URL, purgatory, waiting *CrawlerQueue) {

	var uri *url.URL
	var img *url.URL
	var robots *robotstxt.RobotsData
	var robotsGroup *robotstxt.Group

	httpClient := utils.NewHttpClient(userAgent)

	robotsUrl, err := url.Parse(domain.String())
	robotsUrl.Path = path.Join(domain.Path, "robots.txt")

	robotsText, err := httpClient.RetrieveContent(robotsUrl.String())

	if err != nil {
		log.Printf("Couldn't retrieve robots.txt from [%v] Falling back to all access mode.", robotsUrl.String())
		robots, _ = robotstxt.FromString("User-agent: *\nAllow: /")

	} else {
		robots, err = robotstxt.FromString(robotsText)

		if err != nil {
			log.Fatalln(err)
		}
	}

	robotsGroup = robots.FindGroup(userAgent)

	for {

		select {
		case uri = <-purgatory.Sites:
			val, err := db.Get(uri.String())

			if err != nil {
				panic(err)
			}

			if robotsGroup.Test(uri.String()) == false {
				log.Printf("Skipping site [%v]. It's forbidden by robots.txt", uri.String())
			} else if val == "" {
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

			if robotsGroup.Test(uri.String()) == false {
				log.Printf("Skipping image [%v]. It's forbidden by robots.txt", uri.String())
			} else if val == "" {
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
	var flUserAgent = flag.String("user-agent", "mng-rdr", "UserAgent for HTTP HEADER")

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
		go crawl.Crawl(i+1, *flUserAgent, waiting.Sites, processed.Sites, content)
	}

	for i := 0; i < nDownloaders; i++ {
		go crawl.Download(i+1, *flUserAgent, *flDir, waiting.Images, processed.Images)
	}

	go ManageQueues(db, *flUserAgent, domain, purgatory, waiting)

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
