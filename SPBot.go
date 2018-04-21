package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Syfaro/telegram-bot-api"
	"github.com/robfig/cron"
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
	var ptgUpdates = new(tgbotapi.UpdatesChannel)
	tgUpdates := *ptgUpdates
	if config.Bots.Telegram.TgWebhook == "" {
		// Initialize polling
		tgBot.RemoveWebhook()
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		tgUpdates, _ = tgBot.GetUpdatesChan(u)
	} else {
		// Initialize webhook & channel for update from API
		tgConURI := config.Bots.Telegram.TgWebhook + ":" + strconv.Itoa(config.Bots.Telegram.TgPort) + "/"
		_, err = tgBot.SetWebhook(tgbotapi.NewWebhook(tgConURI + tgBot.Token))
		if err != nil {
			log.Fatal(err)
		}
		// Listen Webhook
		tgUpdates = tgBot.ListenForWebhook("/" + tgBot.Token)
		go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(config.Bots.Telegram.TgPort), nil)
	}
	// Cron for subscriptions
	c := cron.New()
	c.AddFunc("0 20 * * * *", func() {
		tg20Msg := tgbotapi.NewMessage(474165300, startMsgText)
		tg20Msg.ParseMode = "Markdown"
		tgBot.Send(tg20Msg)
	})
	c.AddFunc("@hourly", func() {
		tg1hMsg := tgbotapi.NewMessage(474165300, "Ku-Ku")
		tg1hMsg.ParseMode = "Markdown"
		tgBot.Send(tg1hMsg)
	})
	c.Start()

	// Get updates from channels
	for {

		select {
		// Updates from Telegram
		case tgUpdate := <-tgUpdates:
			toOriginal := false
			// Inline keyboard Callback Query handler
			if tgUpdate.CallbackQuery != nil {
				tgBot.AnswerCallbackQuery(tgbotapi.NewCallback(tgUpdate.CallbackQuery.ID, tgUpdate.CallbackQuery.Data))
				tgCbMsg := tgbotapi.NewMessage(tgUpdate.CallbackQuery.Message.Chat.ID, "")
				tgCbMsg.ParseMode = "Markdown"
				switch tgUpdate.CallbackQuery.Data {
				case "help":
					tgCbMsg.Text = helpMsgText
				case "start":
					tgCbMsg.Text = startMsgText
				}
				tgBot.Send(tgCbMsg)
				continue
			}
			//Simple Message Handler
			tgMsg := tgbotapi.NewMessage(tgUpdate.Message.Chat.ID, "")
			tgMsg.ParseMode = "Markdown"
			// If no command say to User
			if !tgUpdate.Message.IsCommand() {
				tgMsg.ReplyToMessageID = tgUpdate.Message.MessageID
				tgMsg.Text = noCmdText
				tgBot.Send(tgMsg)
				continue
			}

			switch tgUpdate.Message.Command() {
			case "help":
				tgMsg.Text = helpMsgText
			case "start":
				tgMsg.Text = startMsgText
			case "subscriptions":
				tgMsg.Text = stubMsgText
				//For inline keyboard
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Help", "help")
				buttonStart := tgbotapi.NewInlineKeyboardButtonData("Start", "start")
				// For keyboard
				// buttonHelp := tgbotapi.NewKeyboardButton("/help")
				// buttonStart := tgbotapi.NewKeyboardButton("/start")

				// var row []tgbotapi.InlineKeyboardButton
				// row = append(row, buttonHelp)
				// row = append(row, buttonBeltsy)
				// keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
				// keyboard := tgbotapi.NewReplyKeyboard(row)

				// For inline keyboard
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonHelp, buttonStart))
				// For keyboard
				// keyboard := tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(buttonHelp, buttonStart))
				// keyboard.OneTimeKeyboard = true
				tgMsg.ReplyMarkup = keyboard
			case "beltsy":
				tgMsg.Text = "[SP](http://esp.md/podrobnosti/2017/07/06/belchane-o-legalizacii-obektov-v-centre-goroda)"
			case "top":
				tgMsg.Text = stubMsgText
			case "news":
				tgMsg.Text = stubMsgText
			case "search":
				tgMsg.Text = stubMsgText
			case "feedback":
				tgMsg.Text = stubMsgText
			case "holidays":
				tgMsg.Text = strconv.Itoa(int(tgUpdate.Message.Chat.ID)) + tgUpdate.Message.Chat.FirstName + time.Unix(int64(tgUpdate.Message.Date), 0).String()
			case "games":
				tgMsg.Text = stubMsgText
			case "donate":
				tgMsg.Text = stubMsgText
			default:
				toOriginal = true
				tgMsg.Text = noCmdText
			}

			if toOriginal {
				tgMsg.ReplyToMessageID = tgUpdate.Message.MessageID
			}
			tgBot.Send(tgMsg)
		default:
		}
	}
}
