package db

import "strings"

func titlesComparator(title1 string, title2 string) (equivalencePercentage int) {
	title1 = strings.ReplaceAll(strings.ToLower(title1), " ", "")
	title2 = strings.ReplaceAll(strings.ToLower(title2), " ", "")

	/*
	   "song nameeeee"
	   "second song name"
	*/

	var start int
	for end := 0; end < len(title1); end++ {
		start++
	}

	equivalencePercentage = 100

	return equivalencePercentage
}
