package main

import (
	"encoding/json"
	"log"
	"os"
)

// Model represents one NWP model entry within a MetadataJSON document.
type Model struct {
	Name            string   `json:"name"`
	DisplayCategory int      `json:"displayCategory"`
	DisplayOrder    int      `json:"displayOrder"`
	DisplayText     string   `json:"displayText"`
	DocType         string   `json:"docType"`
	FcstLens        []int    `json:"fcstLens"`
	Maxdate         int      `json:"maxdate"`
	Mindate         int      `json:"mindate"`
	Model           string   `json:"model"`
	Numrecs         int      `json:"numrecs"`
	Regions         []string `json:"regions"`
	Subset          string   `json:"subset"`
	Thresholds      []string `json:"thresholds"`
	Variables       []string `json:"variables"`
	Type            string   `json:"type"`
	Version         string   `json:"version"`
}

// MetadataJSON is the Couchbase document written at key MD:matsGui:<name>:COMMON:V01.
type MetadataJSON struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	App       string  `json:"app"`
	Type      string  `json:"type"`
	Version   string  `json:"version"`
	Subset    string  `json:"subset"`
	DocType   string  `json:"docType"`
	Generated bool    `json:"generated"`
	Updated   int     `json:"updated"`
	Models    []Model `json:"models"`
}

// init runs before main() is evaluated
func init() {
	log.Println("write-to-db:init()")
}

// writeMetadata writes metadata to a local file when path is non-empty, otherwise upserts to Couchbase.
func writeMetadata(conn CbConnection, metadata MetadataJSON, path string) {
	if (path != "") {
		log.Println("writeMetadataToFile(" + path + ")")
		writeStructToFile(metadata, path)
		return
	} else {
		log.Println("writeMetadataToDb()")
		_, err := conn.Collection.Upsert(metadata.ID, metadata, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func writeStructToFile(metadata MetadataJSON, path string) {
	log.Println("writeMetadataToFile(" + path + ")")
	jsonData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile(path, jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
