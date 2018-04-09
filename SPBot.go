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
	// Connect to bot
	config, err := LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	bot, err := tgbotapi.NewBotAPI(config.Bots.Telegram.TgApikey)
	if err != nil {
		log.Panic(err)
	}
	// TODO: Next 2 strings for development may remove in production
	bot.Debug = true
	fmt.Printf("Hello, I am %s", bot.Self.UserName)
	// Initialize webhook & channel for update from API
	conURI := config.Bots.Telegram.TgWebhook + ":" + strconv.Itoa(config.Bots.Telegram.TgPort) + "/"
	_, err = bot.SetWebhook(tgbotapi.NewWebhook(conURI + bot.Token))
	if err != nil {
		log.Fatal(err)
	}

	noCmdText := `Извините, это не похоже на комманду. Попробуйте набрать "/help" для просмотра доступных комманд`
	updates := bot.ListenForWebhook("/" + bot.Token)
	go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(config.Bots.Telegram.TgPort), nil)
	// Get updates from channel
	for {
		select {
		case update := <-updates:
			ChatID := update.Message.Chat.ID
			MessageID := update.Message.MessageID
			Text := update.Message.Text
			// UserId := update.Message.From.ID
			// UserName := update.Message.From.UserName
			// FirstName := update.Message.From.FirstName
			// LastName := update.Message.From.LastName
			// noCmdMsg := tgbotapi.NewMessage(ChatID, noCmdText)
			// toOriginal := false
			msg := tgbotapi.NewMessage(ChatID, "")
			msg.ParseMode = "Markdown"

			if !strings.HasPrefix(Text, "/") {
				msg.ReplyToMessageID = MessageID
				msg.Text = noCmdText
				bot.Send(msg)
				continue
			}
		}
	}
}
