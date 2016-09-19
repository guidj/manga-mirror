package utils

import "regexp"

//IsValidUrl checks if a string is a valid URL
func IsValidUrl(url string) (valid bool, err error) {

	pattern := "(@)?(href=')?(HREF=')?(HREF=\")?(href=\")?(http[s]?://)?[a-zA-Z_0-9\\-]+(\\.\\w[a-zA-Z_0-9\\-]+)+(/[#&\\n\\-=?\\+\\%/\\.\\w]+)?"

	valid, err = regexp.MatchString(pattern, url)

	return
}
