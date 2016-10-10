Manga-Mirror
==============

mangareader.net crawler


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

```


```
./bin/manga-mirror -url [some-manga-url] -filter [some-regex-pattern]
```
