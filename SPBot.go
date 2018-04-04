package main

import (
	"encoding/json"
	"log"
	"os"
)

// APIKeys API keys from provider
type APIKeys struct {
	TgbAPI string `json:"tgb"`
}

// LoadAPIKeys load API keys from json file
func LoadAPIKeys(file string) (APIKeys, error) {
	var apikeys APIKeys
	apikeysFile, err := os.Open(file)
	defer apikeysFile.Close()
	if err != nil {
		log.Panic(err)
	}
	jsonParse := json.NewDecoder(apikeysFile)
	err = jsonParse.Decode(&apikeys)
	return apikeys, err
}

func main() {}
