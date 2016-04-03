Manga-Mirror
==============

mangareader.net crawler


Usage
----------

To use the crawler, we need to specify two parameters:

  Url: the web address of the manga we want to download
  Keywords: a list of keywords that should appear in the URL of the manga images

Keywords are required because not all images on the site will be from the manga, so the keywords tell the crawler what images we're interested in.


./manga-mirror "http://web-address-of-manga" "keyword1,keyword2"

e.g.

./manga-mirror "http://mangareader.net/naruto" "naruto"
