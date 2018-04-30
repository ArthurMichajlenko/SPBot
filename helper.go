package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Syfaro/telegram-bot-api"
)

// Config bots configurations
type Config struct {
	Bots         Bots   `json:"bots"`
	FileHolidays string `json:"file_holidays"`
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

//Holidays holidays
type Holidays struct {
	Day     string
	Month   string
	Holiday string
	Date    time.Time
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

// LoadHolidays returns holidays reading from file
func LoadHolidays(file string) ([]Holidays, error) {
	var holidays []Holidays
	var holiday Holidays
	var row []string
	holidaysFile, err := os.Open(file)
	defer holidaysFile.Close()
	if err != nil {
		return holidays, err
	}
	scanner := bufio.NewScanner(holidaysFile)
	for scanner.Scan() {
		row = strings.Split(scanner.Text(), "|")
		holiday.Day = row[0]
		year, _, _ := time.Now().Date()
		loc := time.Now().Location()
		mon, _ := strconv.Atoi(row[1])
		day, _ := strconv.Atoi(row[0])
		holiday.Date = time.Date(year, time.Month(mon), day, 0, 0, 0, 0, loc)
		switch row[1] {
		case "01":
			holiday.Month = "Январь"
		case "02":
			holiday.Month = "Февраль"
		case "03":
			holiday.Month = "Март"
		case "04":
			holiday.Month = "Апрель"
		case "05":
			holiday.Month = "Май"
		case "06":
			holiday.Month = "Июнь"
		case "07":
			holiday.Month = "Июль"
		case "08":
			holiday.Month = "Август"
		case "09":
			holiday.Month = "Сентябрь"
		case "10":
			holiday.Month = "Октябрь"
		case "11":
			holiday.Month = "Ноябрь"
		case "12":
			holiday.Month = "Декабрь"
		default:
			holiday.Month = ""
		}
		holiday.Holiday = row[2]
		holidays = append(holidays, holiday)
	}
	return holidays, err
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
