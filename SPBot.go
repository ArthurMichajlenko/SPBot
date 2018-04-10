package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Syfaro/telegram-bot-api"
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
	FbPort     int    `json:"fb_port"`
	FbPathCERT string `json:"fb_path_cert"`
}

// Telegram bot configuration
type Telegram struct {
	TgApikey   string `json:"tg_apikey"`
	TgWebhook  string `json:"tg_webhook"`
	TgPort     int    `json:"tg_port"`
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
	// Connect to Telegram bot
	tgBot, err := tgbotapi.NewBotAPI(config.Bots.Telegram.TgApikey)
	if err != nil {
		log.Panic(err)
	}
	// TODO: Next 2 strings for development may remove in production
	tgBot.Debug = true
	fmt.Println("Hello, I am", tgBot.Self.UserName)
	// Initialize webhook & channel for update from API
	tgConURI := config.Bots.Telegram.TgWebhook + ":" + strconv.Itoa(config.Bots.Telegram.TgPort) + "/"
	_, err = tgBot.SetWebhook(tgbotapi.NewWebhook(tgConURI + tgBot.Token))
	if err != nil {
		log.Fatal(err)
	}

	noCmdText := `Извините, это не похоже на комманду. Попробуйте набрать "/help" для просмотра доступных комманд`
	tgUpdates := tgBot.ListenForWebhook("/" + tgBot.Token)
	go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(config.Bots.Telegram.TgPort), nil)
	// Get updates from channel
	for {
		select {
		case tgUpdate := <-tgUpdates:
			ChatID := tgUpdate.Message.Chat.ID
			MessageID := tgUpdate.Message.MessageID
			Text := tgUpdate.Message.Text
			// UserId := tgUpdate.Message.From.ID
			// UserName := tgUpdate.Message.From.UserName
			// FirstName := tgUpdate.Message.From.FirstName
			// LastName := tgUpdate.Message.From.LastName
			// noCmdMsg := tgbotapi.NewMessage(ChatID, noCmdText)
			// toOriginal := false
			msg := tgbotapi.NewMessage(ChatID, "")
			msg.ParseMode = "Markdown"

			if !strings.HasPrefix(Text, "/") {
				msg.ReplyToMessageID = MessageID
				msg.Text = noCmdText
				tgBot.Send(msg)
				continue
			}
		}
	}
}
