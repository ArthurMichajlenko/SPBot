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

	tgbotapi "github.com/Syfaro/telegram-bot-api"
	"github.com/jordan-wright/email"
	"github.com/mileusna/viber"
)

var isCarousel bool

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
	SMTPServer string `json:"smtp_server"`
	SMTPPort   string `json:"smtp_port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	EmailFrom  string `json:"email_from"`
	EmailTo    string `json:"email_to"`
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
	email.To = append(email.To, botConfig.Feedback.Email.EmailTo)
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

// msgReceived will be called everttime when user send a message
func msgReceived(v *viber.Viber, u viber.User, m viber.Message, token uint64, t time.Time) {
	noCmdText := `Извините, я не понял. Попробуйте набрать "help"`
	stubMsgText := ` Извините, пока не реализовано`
	startMsgText := `Добро пожаловать! Предлагаем Вам подписаться на новости на сайте "СП". Вы сможете настроить рассылку так, как Вам удобно.`
	helpMsgText := `Что я умею:
	help - выводит это сообщение.
	start - подключение к боту.
	subscriptions - управление Вашими подписками.
	alerts - городские оповещения.
	top - самое популярное в "СП".
	news - последние материалы на сайте "СП".
	search - поиск по сайту "СП".
	feedback - задать вопрос/сообщить новость.
	holidays - календарь праздников.
	games - игры.
	donate - поддержать "СП".`
	startMsgEndText := `Спасибо за Ваш выбор! Вы можете отписаться от нашей рассылки в любой момент в меню "subscriptions".
	Взгляните на весь список команд, с помощью которых Вы можете управлять возможностями нашего бота.` + "\n" + helpMsgText
	kb := v.NewKeyboard("", true)
	kb.DefaultHeight = false
	btHelp := v.NewTextButton(6, 1, "reply", "help", `<font color="#ffffff">Help</font>`)
	btHelp.SetBgColor("#752f35")
	btDonate := v.NewTextButton(2, 1, "reply", "donate", `<font color="#ffffff">Donate</font>`)
	btDonate.SetBgColor("#752f35")
	btSearch := v.NewTextButton(2, 1, "reply", "search", `<font color="#ffffff">Search</font>`)
	btSearch.SetBgColor("#752f35")
	btFeedback := v.NewTextButton(2, 1, "reply", "feedback", `<font color="#ffffff">Feedback</font>`)
	btFeedback.SetBgColor("#752f35")
	btGames := v.NewTextButton(2, 1, "reply", "games", `<font color="#ffffff">Games</font>`)
	btGames.SetBgColor("#752f35")
	btHolidays := v.NewTextButton(2, 1, "reply", "holidays", `<font color="#ffffff">Holidays</font>`)
	btHolidays.SetBgColor("#752f35")
	btAlerts := v.NewTextButton(2, 1, "reply", "alerts", `<font color="#ffffff">Alerts</font>`)
	btAlerts.SetBgColor("#752f35")
	btNews := v.NewTextButton(2, 1, "reply", "news", `<font color="#ffffff">News</font>`)
	btNews.SetBgColor("#752f35")
	btTop := v.NewTextButton(2, 1, "reply", "top", `<font color="#ffffff">Top</font>`)
	btTop.SetBgColor("#752f35")
	btSubscriptions := v.NewTextButton(2, 1, "reply", "subscriptions", `<font color="#ffffff">Subscriptions</font>`)
	btSubscriptions.SetBgColor("#752f35").TextSizeSmall()
	kb.AddButton(btNews)
	kb.AddButton(btAlerts)
	kb.AddButton(btTop)
	kb.AddButton(btSearch)
	kb.AddButton(btHolidays)
	kb.AddButton(btGames)
	kb.AddButton(btSubscriptions)
	kb.AddButton(btFeedback)
	kb.AddButton(btDonate)
	kb.AddButton(btHelp)
	switch m.(type) {
	case *viber.TextMessage:
		msg := v.NewTextMessage("")
		txt := strings.ToLower(m.(*viber.TextMessage).Text)
		switch txt {
		case "help":
			isCarousel = false
			msg = v.NewTextMessage(helpMsgText)
		case "start":
			isCarousel = false
			msg = v.NewTextMessage(startMsgText + "\n" + startMsgEndText)
		case "subscriptions":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		case "alerts":
			isCarousel = true
			msgCarouselCity := v.NewRichMediaMessage(6, 7, "#752f35")
			var city News
			numPage := 0
			urlCity := botConfig.QueryCityDisp
			city, err := NewsQuery(urlCity, numPage)
			if err != nil {
				log.Println(err)
			}
			v.SendTextMessage(u.ID, "Городские оповещения")
			for _, cityItem := range city.Nodes {
				msgCarouselCity.AddButton(v.NewTextButton(6, 2, viber.OpenURL, cityItem.Node.NodePath, cityItem.Node.NodeDate+"\n"+cityItem.Node.NodeTitle))
				msgCarouselCity.AddButton(v.NewImageButton(6, 4, viber.OpenURL, cityItem.Node.NodePath, cityItem.Node.NodeCover["src"]))
				msgCarouselCity.AddButton(v.NewTextButton(6, 1, viber.OpenURL, cityItem.Node.NodePath, `<font color="#ffffff>Подробнее...</font>`).SetBgColor("#752f35"))
			}
			urlCity = botConfig.QueryCityAfisha
			city, err = NewsQuery(urlCity, numPage)
			for _, cityItem := range city.Nodes {
				msgCarouselCity.AddButton(v.NewTextButton(6, 2, viber.OpenURL, cityItem.Node.NodePath, cityItem.Node.NodeDate+"\n"+cityItem.Node.NodeTitle))
				msgCarouselCity.AddButton(v.NewImageButton(6, 4, viber.OpenURL, cityItem.Node.NodePath, cityItem.Node.NodeCover["src"]))
				msgCarouselCity.AddButton(v.NewTextButton(6, 1, viber.OpenURL, cityItem.Node.NodePath, `<font color="#ffffff>Подробнее...</font>`).SetBgColor("#752f35"))
			}
			v.SendMessage(u.ID, msgCarouselCity)
		case "top":
			isCarousel = true
			msgCarouselView := v.NewRichMediaMessage(6, 7, "#752f35")
			msgCarouselComment := v.NewRichMediaMessage(6, 7, "#752f35")
			var top News
			urlTop := botConfig.QueryTopViews
			top, err := NewsQuery(urlTop, -1)
			if err != nil {
				log.Println(err)
			}
			v.SendTextMessage(u.ID, "Самые читаемые")
			for _, topItem := range top.Nodes {
				msgCarouselView.AddButton(v.NewTextButton(6, 2, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeDate+"\n"+topItem.Node.NodeTitle))
				msgCarouselView.AddButton(v.NewImageButton(6, 4, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeCover["src"]))
				msgCarouselView.AddButton(v.NewTextButton(6, 1, viber.OpenURL, topItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor("#752f35"))
			}
			v.SendMessage(u.ID, msgCarouselView)
			urlTop = botConfig.QueryTopComments
			top, err = NewsQuery(urlTop, -1)
			if err != nil {
				log.Println(err)
			}
			v.SendTextMessage(u.ID, "Самые комментируемые")
			for _, topItem := range top.Nodes {
				msgCarouselComment.AddButton(v.NewTextButton(6, 2, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeDate+"\n"+topItem.Node.NodeTitle))
				msgCarouselComment.AddButton(v.NewImageButton(6, 4, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeCover["src"]))
				msgCarouselComment.AddButton(v.NewTextButton(6, 1, viber.OpenURL, topItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor("#752f35"))
			}
			v.SendMessage(u.ID, msgCarouselComment)
		case "news":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		case "search":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		case "feedback":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		case "holidays":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		case "games":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		case "donate":
			isCarousel = false
			msg = v.NewTextMessage(txt + stubMsgText)
		default:
			if !isCarousel {
				msg = v.NewTextMessage(noCmdText)
			} else {
				isCarousel = false
			}
		}
		msg.SetKeyboard(kb)
		v.SendMessage(u.ID, msg)
	case *viber.URLMessage:
		url := m.(*viber.URLMessage).Media
		v.SendTextMessage(u.ID, "You send me this URL:"+url)
	case *viber.PictureMessage:
		v.SendTextMessage(u.ID, "Nice pic")
	}
}
