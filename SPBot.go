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
	// Standart messages
	noCmdText := `Извините, я не понял. Попробуйте набрать "/help"`
	stubMsgText := `_Извините, пока не реализовано_`
	startMsgText := `Здравствуйте! Подключайтесь к новостному боту "СП"-умному ассистенту, который поможет Вам получать полезную и важную информацию в телефоне удобным для Вас образом.
	Чтобы посмотреть, что я умею наберите "/help"`
	helpMsgText := `Что я умею:
	/help - выводит это сообщение.
	/start - подключение к боту.
	/subscriptions - управление Вашими подписками.
	/beltsy - городские новости и уведомления.
	/top - самое популярное в "СП".
	/news - последние материалы на сайте "СП".
	/search - поиск по сайту "СП".
	/feedback - задать вопрос/сообщить новость.
	/holidays - календарь праздников.
	/games - поиграть в игру.
	/donate - поддержать "СП".`
	// Listen Webhook
	tgUpdates := tgBot.ListenForWebhook("/" + tgBot.Token)
	go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(config.Bots.Telegram.TgPort), nil)
	// Get updates from channels
	for {
		select {
		// Updates from Telegram
		case tgUpdate := <-tgUpdates:
			toOriginal := false
			tgMsg := tgbotapi.NewMessage(tgUpdate.Message.Chat.ID, "")
			tgMsg.ParseMode = "Markdown"
			// If no command say to User
			if !strings.HasPrefix(tgUpdate.Message.Text, "/") {
				tgMsg.ReplyToMessageID = tgUpdate.Message.MessageID
				tgMsg.Text = noCmdText
				tgBot.Send(tgMsg)
				continue
			}

			switch strings.ToLower(strings.Split(tgUpdate.Message.Text, " ")[0]) {
			case "/help":
				tgMsg.Text = helpMsgText
			case "/start":
				tgMsg.Text = startMsgText
			case "/subscriptions":
				tgMsg.Text = "[SP](http://esp.md/sobytiya/2018/04/16/uznay-gde-ty-dolzhen-golosovat-na-vyborah-primara-belc)"
			case "/beltsy":
				tgMsg.Text = stubMsgText
			case "/top":
				tgMsg.Text = stubMsgText
			case "/news":
				tgMsg.Text = stubMsgText
			case "/search":
				tgMsg.Text = stubMsgText
			case "/feedback":
				tgMsg.Text = stubMsgText
			case "/holidays":
				tgMsg.Text = stubMsgText
			case "/games":
				tgMsg.Text = stubMsgText
			case "/donate":
				tgMsg.Text = stubMsgText
			default:
				toOriginal = true
				tgMsg.Text = noCmdText
			}

			if toOriginal {
				tgMsg.ReplyToMessageID = tgUpdate.Message.MessageID
			}
			tgBot.Send(tgMsg)
		}
	}
}
