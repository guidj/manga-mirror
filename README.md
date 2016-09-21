Manga-Mirror
==============

mangareader.net crawler


Usage
----------
Parameters:

```
  -database, -db [string]
	   Path for crawler sync index (default "_mrdb")
  -directory, -dir [string]
    	Path to store downlaoded media (default "_media")
  -domain, -url [string]
    	Web URL to crawl for media
  -filter-regex, -f [string]
    	Regex pattern to filter URIs, e.g. 'mangareader.net|naruto'
```


```
./bin/manga-mirror -url [some-manga-url] -f [some-regex-pattern]
```
