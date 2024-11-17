package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

func printJSON(data any) {
	jsonData, _ := json.MarshalIndent(data, "", "\t")

	fmt.Println(string(jsonData))
}

func PrintJSONs(sep string, data ...any) {
	for i, d := range data {
		printJSON(d)
		if i > 0 || sep != "" {
			fmt.Println(sep)
		}
	}
}

func GetTYVidIDFromURL(url string) string {
	start := strings.Index(url, "?v=")
	end := strings.Index(url, "&")
	if end == -1 {
		return url[start+3:]
	}
	return url[start+3 : end]

}
