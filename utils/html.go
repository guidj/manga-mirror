package utils

import (
	"regexp"
)

// IsValidURL checks if a string is a valid URL.
func IsValidURL(url string) (bool, error) {
	pattern := "(@)?(href=')?(HREF=')?(HREF=\")?(href=\")?(http[s]?://)?[a-zA-Z_0-9\\-]+(\\.\\w[a-zA-Z_0-9\\-]+)+(/[#&\\n\\-=?\\+\\%/\\.\\w]+)?"
	valid, err := regexp.MatchString(pattern, url)
	return valid, err
}

// MakeRegexMatcher returns a function to check if any string matches the given regex.
func MakeRegexMatcher(regex string) func(text string) bool {
	re := regexp.MustCompile(regex)
	return func(text string) bool {
		return re.MatchString(text)
	}
}
