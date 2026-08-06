package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func getTabbedString(count int) (rv string) {
	rv = ""
	for i := 0; i < count; i++ {
		rv = rv + "\t"
	}
	return rv
}

func printStringArray(in []string) {
	for i := 0; i < len(in); i++ {
		fmt.Println(in[i])
	}
}

func jsonPrettyPrint(in []interface{}) string {
	jsonText, err := json.Marshal(in)
	if err != nil {
		fmt.Println("ERROR PROCESSING STREAMING OUTPUT:", err)
	}
	var out bytes.Buffer
	json.Indent(&out, jsonText, "", "\t")
	return out.String()
}

func jsonPrettyPrintStruct(in interface{}) string {
	jsonText, err := json.Marshal(in)
	if err != nil {
		fmt.Println("ERROR PROCESSING STREAMING OUTPUT:", err)
	}
	var out bytes.Buffer
	json.Indent(&out, jsonText, "", "\t")
	return out.String()
}

// ConvertSlice reinterprets a []any as []E using direct type assertions.
// It panics if any element is not assignable to E.
func ConvertSlice[E any](in []any) (out []E) {
	out = make([]E, 0, len(in))
	for _, v := range in {
		out = append(out, v.(E))
	}
	return
}
