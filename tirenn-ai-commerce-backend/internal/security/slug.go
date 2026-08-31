package security

import (
	"regexp"
	"strings"
)

// Slugify converts text into a clean URL-safe slug
func Slugify(text string) string {
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	slug := strings.ToLower(reg.ReplaceAllString(text, "-"))
	return strings.Trim(slug, "-")
}
