package utils

import (
	"encoding/json"
	"fmt"
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
