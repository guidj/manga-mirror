Manga-Mirror
==============

[mangareader.net](http://www.mangareader.net/) crawler.

Build
----------
```
$ make
```

It builds an executable under `$GOPATH/bin/` named `manga-mirror`.


Usage
----------
Parameters:

```
  -db [string]
	   Path for crawler sync index (default "_mrdb")
  -dir [string]
    	Path to store downlaoded media (default "_media")
  -url [string]
    	Web URL to crawl for media
  -filter [string]
    	Regex pattern to filter URIs, e.g. 'mangareader.net|naruto'
  -log [string]
    	log file name, for logging (default STDOUT)
  -user-agent string
    	UserAgent for HTTP HEADER (default "mng-rdr")
```


```
$ manga-mirror -url [some-manga-url] -filter [some-regex-pattern]
```


Misc
-----------

This crawler can be used to crawl other sites, and download images. Though it is ever hardly used or tested at that task.
Note that this crawler respects `robots.txt` configurations on which paths can be crawled, but not on the crawling interval.
