package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// Config bots configurations
type Config struct {
	Bots Bots `json:"bots"`
}

// Bots configuration webhook,port,APIkey etc.
type Bots struct {
	Telegram Telegram `json:"telegram"`
	Facebook Facebook `json:"facebook"`
}

// Facebook bot configuration
type Facebook struct {
	FbApikey   string `json:"fb_apikey"`
	FbWebhook  string `json:"fb_webhook"`
	FbPort     int64  `json:"fb_port"`
	FbPathCERT string `json:"fb_path_cert"`
}

// Telegram bot configuration
type Telegram struct {
	TgApikey   string `json:"tg_apikey"`
	TgWebhook  string `json:"tg_webhook"`
	TgPort     int64  `json:"tg_port"`
	TgPathCERT string `json:"tg_path_cert"`
}

// LoadConfigBots returns config reading from json file
func LoadConfigBots(file string) (Config, error) {
	var botsconfig Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		log.Panic(err)
	}
	jsonParse := json.NewDecoder(configFile)
	err = jsonParse.Decode(&botsconfig)
	if err != nil {
		log.Panic(err)
	}
	return botsconfig, err
}

func main() {
	config, err := LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("%v", config)
}
