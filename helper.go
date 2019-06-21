package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jordan-wright/email"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

// Config bots configurations.
type Config struct {
	Bots             Bots     `json:"bots"`
	Feedback         Feedback `json:"feedback"`
	FileHolidays     string   `json:"file_holidays"`
	QueryTopViews    string   `json:"query_top_views"`
	QueryTopComments string   `json:"query_top_comments"`
	QuerySearch      string   `json:"query_search"`
	QueryNews1H      string   `json:"query_news_1h"`
	QueryNews24H     string   `json:"query_news_24h"`
	QueryCityDisp    string   `json:"query_city_disp"`
	QueryCityAfisha  string   `json:"query_city_afisha"`
	QueryGames       string   `json:"query_games"`
	Debug            bool     `json:"debug"`
}

// Bots configuration webhook,port,APIkey etc.
type Bots struct {
	Telegram Telegram `json:"telegram"`
	Facebook Facebook `json:"facebook"`
	Viber    Viber    `json:"viber"`
}

// Facebook bot configuration.
type Facebook struct {
	FbApikey   string `json:"fb_apikey"`
	FbWebhook  string `json:"fb_webhook"`
	FbPort     int    `json:"fb_port"`
	FbPathCERT string `json:"fb_path_cert"`
	FbPathKey  string `json:"fb_path_key"`
	LogFile    string `json:"log_file"`
}

// Telegram bot configuration.
type Telegram struct {
	TgApikey   string `json:"tg_apikey"`
	TgWebhook  string `json:"tg_webhook"`
	TgPort     int    `json:"tg_port"`
	TgPathCERT string `json:"tg_path_cert"`
	TgPathKey  string `json:"tg_path_key"`
	LogFile    string `json:"log_file"`
}

// Viber bot configuration.
type Viber struct {
	VBApikey   string `json:"vb_apikey"`
	VBWebhook  string `json:"vb_webhook"`
	VBPort     int    `json:"vb_port"`
	VBPathCERT string `json:"vb_path_cert"`
	VBPathKey  string `json:"vb_path_key"`
	LogFile    string `json:"log_file"`
}

// Feedback botConfig for feedback.
type Feedback struct {
	Email Email `json:"email"`
}

// Email botConfig email parameters.
type Email struct {
	SMTPServer string   `json:"smtp_server"`
	SMTPPort   string   `json:"smtp_port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	EmailFrom  string   `json:"email_from"`
	EmailTo    []string `json:"email_to"`
}

// News from query esp.md.
type News struct {
	Nodes []NodeElement `json:"nodes"`
}

// NodeElement from news.
type NodeElement struct {
	Node NodeNews `json:"node"`
}

// NodeNews what is in node.
type NodeNews struct {
	NodeID    string            `json:"node_id"`
	NodeTitle string            `json:"title"`
	NodeBody  string            `json:"node_body"`
	NodeCover map[string]string `json:"node_cover"`
	NodePath  string            `json:"node_path"`
	NodeDate  string            `json:"node_date"`
}

//Holidays holidays.
type Holidays struct {
	Day     string
	Month   string
	Holiday string
	Date    time.Time
}

//AttachFile properties attached file
type AttachFile struct {
	FileName    []string
	ContentType []string
	BotFile     tgbotapi.File
}

//TgUser Telegram User. BoltDb
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

//TgMessageOwner info about who send message.
type TgMessageOwner struct {
	ID        string
	Username  string
	FirstName string
	LastName  string
}

// LoadHolidays returns holidays reading from file.
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
			holiday.Month = "января"
		case "02":
			holiday.Month = "февраля"
		case "03":
			holiday.Month = "марта"
		case "04":
			holiday.Month = "апреля"
		case "05":
			holiday.Month = "мая"
		case "06":
			holiday.Month = "июня"
		case "07":
			holiday.Month = "июля"
		case "08":
			holiday.Month = "августа"
		case "09":
			holiday.Month = "сентября"
		case "10":
			holiday.Month = "октября"
		case "11":
			holiday.Month = "ноября"
		case "12":
			holiday.Month = "декабря"
		default:
			holiday.Month = ""
		}
		holiday.Holiday = row[2]
		holidays = append(holidays, holiday)
	}
	return holidays, err
}

//CheckNewsRange check if news date between 24 hours period
func CheckNewsRange(newsDate string) bool {
	layout := "02.01.2006 - 15:04MST"
	t := time.Now()
	zone, _ := t.Zone()
	timeNews, _ := time.Parse(layout, newsDate+zone)
	timeNews = timeNews.Local()
	return timeNews.After(t.Add(-time.Hour*24)) && timeNews.Before(t)
}

// LoadConfigBots returns botConfig reading from json file.
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

//SubButtons create keyboard for subscriptions.
func SubButtons(update *tgbotapi.Update, user *TgUser) tgbotapi.EditMessageReplyMarkupConfig {
	bt9 := "Утром"
	bt20 := "Вечером"
	btL := "Последние новости"
	btT := "Самое популярное"
	btC := "Городские оповещения"
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

// NewsQuery get Nodes from esp.md. -1 without page
func NewsQuery(url string, numPage int) (News, error) {
	var news News
	if numPage != -1 {
		url += strconv.Itoa(numPage)
	}
	res, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}
	defer res.Body.Close()
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal(r, &news)
	return news, err
}

// SearchQuery get Nodes from esp.md.
func SearchQuery(query string, numPage int) (News, error) {
	var search News
	queryURL := botConfig.QuerySearch + url.QueryEscape(query) + "&page=" + strconv.Itoa(numPage)
	res, err := http.Get(queryURL)
	if err != nil {
		log.Println(err)
	}
	defer res.Body.Close()
	r, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
	}
	err = json.Unmarshal(r, &search)
	return search, err
}

// SendFeedback sends email feedback.
func SendFeedback(subject string, text string, attachmentURLs []string, fileName []string, contentType []string) error {
	// Create email auth
	botConfig, err := LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	smtpAuth := smtp.PlainAuth("", botConfig.Feedback.Email.Username, botConfig.Feedback.Email.Password, botConfig.Feedback.Email.SMTPServer)
	email := email.NewEmail()
	email.From = botConfig.Feedback.Email.EmailFrom
	email.To = botConfig.Feedback.Email.EmailTo
	email.Subject = subject
	email.Text = []byte(text)
	if attachmentURLs == nil {
		return email.Send(botConfig.Feedback.Email.SMTPServer+":"+botConfig.Feedback.Email.SMTPPort, smtpAuth)
	}
	for i, attachmentURL := range attachmentURLs {
		res, err := http.Get(attachmentURL)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		_, err = email.Attach(res.Body, fileName[i], contentType[i])
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
	}
	return email.Send(botConfig.Feedback.Email.SMTPServer+":"+botConfig.Feedback.Email.SMTPPort, smtpAuth)
}
