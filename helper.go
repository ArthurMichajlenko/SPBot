package main

import (
	"encoding/json"
	"log"
	"os"

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

//TgUser Telegram User
type TgUser struct {
	ChatID            int64 `storm:"id"`
	FirstName         string
	LastName          string
	Username          string `storm:"unique"`
	LastDate          int
	Subscribe9        bool
	Subscribe20       bool
	SubscribeLast     bool
	SubscribeCity     bool
	SubscribeTop      bool
	SubscribeHolidays bool
	RssLastID         int
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

//SubButtons create keyboard for subscriptions
func SubButtons(update *tgbotapi.Update, user *TgUser) tgbotapi.EditMessageReplyMarkupConfig {
	bt9 := "Утром"
	bt20 := "Вечером"
	btL := "Последние новости"
	btT := "Самое популярное"
	btC := "Городские уведомления"
	btH := "Календарь праздников"
	btF := "Продолжить..."
	if user.Subscribe9 {
		bt9 = "\u2705" + bt9
	}
	if user.Subscribe20 {
		bt20 = "\u2705" + bt20
	}
	if user.SubscribeLast {
		btL = "\u2705" + btL
	}
	if user.SubscribeTop {
		btT = "\u2705" + btT
	}
	if user.SubscribeCity {
		btC = "\u2705" + btC
	}
	if user.SubscribeHolidays {
		btH = "\u2705" + btH
	}
	buttonSubscribe9 := tgbotapi.NewInlineKeyboardButtonData(bt9, "subscribe9")
	buttonSubscribe20 := tgbotapi.NewInlineKeyboardButtonData(bt20, "subscribe20")
	buttonSubscribeLast := tgbotapi.NewInlineKeyboardButtonData(btL, "subscribelast")
	buttonSubscribeTop := tgbotapi.NewInlineKeyboardButtonData(btT, "subscribetop")
	buttonSubscribeCity := tgbotapi.NewInlineKeyboardButtonData(btC, "subscribecity")
	buttonSubscribeHolidays := tgbotapi.NewInlineKeyboardButtonData(btH, "subscribeholidays")
	buttonSubscribeFinish := tgbotapi.NewInlineKeyboardButtonData(btF, "subscribefinish")
	var row0 []tgbotapi.InlineKeyboardButton
	var row1 []tgbotapi.InlineKeyboardButton
	var row2 []tgbotapi.InlineKeyboardButton
	var row3 []tgbotapi.InlineKeyboardButton
	row0 = append(row0, buttonSubscribe9)
	row0 = append(row0, buttonSubscribe20)
	row1 = append(row1, buttonSubscribeLast)
	row1 = append(row1, buttonSubscribeTop)
	row2 = append(row2, buttonSubscribeCity)
	row2 = append(row2, buttonSubscribeHolidays)
	row3 = append(row3, buttonSubscribeFinish)
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row0, row1, row2, row3)
	return tgbotapi.NewEditMessageReplyMarkup(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, keyboard)
}
