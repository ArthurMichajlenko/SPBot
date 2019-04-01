package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asdine/storm"

	"github.com/jordan-wright/email"
	"github.com/mileusna/viber"
)

var spColorBG = "#752f35"

// Page number page of results
var Page int
var (
	isCarousel      bool
	isFeedback      bool
	isSearch        bool
	searchString    string
	emailBody       string
	emailSubject    string
	fileName        []string
	contentType     []string
	attachmentURLs  []string
	attachmentCount int
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

//VbUser Viber User. BoltDb
type VbUser struct {
	ID                string `storm:"unique"`
	Username          string
	LastDate          time.Time
	Subscribe9        bool
	Subscribe20       bool
	SubscribeLast     bool
	SubscribeCity     bool
	SubscribeTop      bool
	SubscribeHolidays bool
}

//VbMessageOwner info about who send message
type VbMessageOwner struct {
	ID       string
	Username string
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

// msgReceived will be called everttime when user send a message
func msgReceived(v *viber.Viber, u viber.User, m viber.Message, token uint64, t time.Time) {
	// noCmdText := `Извините, я не понял. Попробуйте набрать "help"`
	noCmdText := `Главное меню`
	// stubMsgText := ` Извините, пока не реализовано`
	feedbackText := `Введите текст сообщения...
	ВНИМАНИЕ: Обязательно укажите ваше имя, фамилию и номер телефона (без этого сообщение не будет рассмотрено)
	Вы можете добавить к сообщению до 5 файлов (фото и/или видео).`
	startMsgText := `Добро пожаловать! Предлагаем вам подписаться на новости на сайте "СП". Вы сможете настроить рассылку так, как вам удобно.`
	helpMsgText := `Что я умею:
	Поздоровавшись с ботом (Hi, Hello, Привет) Вы увидите Главное меню
	help - выводит это сообщение (кнопка "Помощь" главного меню).
	start - подключение к боту.
	subscriptions - управление вашими подписками (кнопка "Управление подписками" главного меню).
	alerts - городские оповещения (кнопка "Городские оповещения" главного меню).
	top - самое популярное в "СП" (кнопка "Самое популярное" главного меню).
	news - последние материалы на сайте "СП" (кнопка "Последние новости" главного меню).
	search - поиск по сайту "СП" (кнопка "Поиск" главного меню).
	feedback - задать вопрос/сообщить новость (кнопка "Спросить.сообщить новость" главного меню).
	holidays - календарь праздников (кнопка "Календарь праздников" главного меню).
	games - игры (кнопка "Игры" главного меню).
	donate - поддержать "СП" (кнопка "Поддержи "СП"" главного меню).`
	// startMsgEndText := `Спасибо за ваш выбор! Вы можете отписаться от нашей рассылки в любой момент в меню "subscriptions".
	startMsgEndText := `Вы можете отписаться от нашей рассылки в любой момент в меню "subscriptions".`
	// Взгляните на весь список команд, с помощью которых Вы можете управлять возможностями нашего бота.` + "\n" + helpMsgText
	kbMain := v.NewKeyboard("", true)
	kbMain.DefaultHeight = false
	btHelp := v.NewTextButton(6, 1, "reply", "help", `<font color="#ffffff">Помощь</font>`)
	btHelp.SetBgColor(spColorBG)
	btDonate := v.NewTextButton(2, 1, "reply", "donate", `<font color="#ffffff">Поддержи "СП"</font>`)
	btDonate.SetBgColor(spColorBG)
	btSearch := v.NewTextButton(2, 1, "reply", "search", `<font color="#ffffff">Поиск</font>`)
	btSearch.SetBgColor(spColorBG)
	btFeedback := v.NewTextButton(2, 1, "reply", "feedback", `<font color="#ffffff">Спросить/сообщить новость</font>`).TextSizeSmall()
	btFeedback.SetBgColor(spColorBG)
	btGames := v.NewTextButton(2, 1, "reply", "games", `<font color="#ffffff">Игры</font>`)
	btGames.SetBgColor(spColorBG)
	btHolidays := v.NewTextButton(2, 1, "reply", "holidays", `<font color="#ffffff">Календарь праздников</font>`)
	btHolidays.SetBgColor(spColorBG)
	btAlerts := v.NewTextButton(2, 1, "reply", "alerts", `<font color="#ffffff">Городские оповещения</font>`).TextSizeSmall()
	btAlerts.SetBgColor(spColorBG)
	btNews := v.NewTextButton(2, 1, "reply", "news", `<font color="#ffffff">Последние новости</font>`)
	btNews.SetBgColor(spColorBG)
	btTop := v.NewTextButton(2, 1, "reply", "top", `<font color="#ffffff">Самое популярное</font>`)
	btTop.SetBgColor(spColorBG)
	btSubscriptions := v.NewTextButton(2, 1, "reply", "subscriptions", `<font color="#ffffff">Управление подписками</font>`)
	btSubscriptions.SetBgColor(spColorBG).TextSizeSmall()
	kbMain.AddButton(btNews)
	kbMain.AddButton(btAlerts)
	kbMain.AddButton(btTop)
	kbMain.AddButton(btSearch)
	kbMain.AddButton(btHolidays)
	kbMain.AddButton(btGames)
	kbMain.AddButton(btSubscriptions)
	kbMain.AddButton(btFeedback)
	kbMain.AddButton(btDonate)
	kbMain.AddButton(btHelp)
	//Subscribe/Unsubscribe
	subscribe9 := `<font color="#ffffff">Утренний дайджест
	<b>Подписаться</b></font>`
	unsubscribe9 := `<font color="#ffffff">Утренний дайджест
	<b><i>Отписаться</i></b></font>`
	subscribe20 := `<font color="#ffffff">Вечерний дайджест
	<b>Подписаться</b></font>`
	unsubscribe20 := `<font color="#ffffff">Вечерний дайджест
	<b><i>Отписаться</i></b></font>`
	subscribeLast := `<font color="#ffffff">Последние новости
	<b>Подписаться</b></font>`
	unsubscribeLast := `<font color="#ffffff">Последние новости
	<b><i>Отписаться</i></b></font>`
	subscribeCity := `<font color="#ffffff">Городские оповещения
	<b>Подписаться</b></font>`
	unsubscribeCity := `<font color="#ffffff">Городские оповещения
	<b><i>Отписаться</i></b></font>`
	subscribeTop := `<font color="#ffffff">Самое популярное
	<b>Подписаться</b></font>`
	unsubscribeTop := `<font color="#ffffff">Самое популярное
	<b><i>Отписаться</i></b></font>`
	subscribeHolidays := `<font color="#ffffff">Календарь праздников
	<b>Подписаться</b></font>`
	unsubscribeHolidays := `<font color="#ffffff">Календарь праздников
	<b><i>Отписаться</i></b></font>`
	//Bolt
	var vbbuser VbUser
	db, err := storm.Open("vbuser.db")
	if err != nil {
		log.Println(err)
	}
	defer db.Close()
	db.Init(&vbbuser)
	//Received messages loop
	switch m.(type) {
	case *viber.TextMessage:
		msg := v.NewTextMessage("")
		txt := strings.ToLower(m.(*viber.TextMessage).Text)
		switch txt {
		case "help":
			isCarousel = false
			msg = v.NewTextMessage(helpMsgText)
			msg.SetKeyboard(kbMain)
			v.SendMessage(u.ID, msg)
		case "start":
			isCarousel = false
			msg = v.NewTextMessage(startMsgText + "\n" + startMsgEndText)
			kbStart := v.NewKeyboard("", false)
			kbStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "substart", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG))
			kbStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG))
			msg.SetKeyboard(kbStart)
			v.SendMessage(u.ID, msg)
		case "substart":
			isCarousel = false
			kbSubStart := v.NewKeyboard("", false)
			db.One("ID", u.ID, &vbbuser)
			if vbbuser.Subscribe9 {
				kbSubStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscr9", unsubscribe9).SetBgColor(spColorBG))
			} else {
				kbSubStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscr9", subscribe9).SetBgColor(spColorBG))
			}
			if vbbuser.Subscribe20 {
				kbSubStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscr20", unsubscribe20).SetBgColor(spColorBG))
			} else {
				kbSubStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscr20", subscribe20).SetBgColor(spColorBG))
			}
			if vbbuser.SubscribeLast {
				kbSubStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscrl", unsubscribeLast).SetBgColor(spColorBG))
			} else {
				kbSubStart.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscrl", subscribeLast).SetBgColor(spColorBG))
			}
			msg = v.NewTextMessage("Выберите подписку:" + "\n" + " Утром -получать дайджест за сутки утром - в 9:00" + "\n" + " Вечером -получать дайджест за сутки вечером - в 20:00" + "\n" + " Последние новости -получать новости сразу по мере их публикации (сообщения будут приходить часто)")
			msg.SetKeyboard(kbSubStart)
			v.SendMessage(u.ID, msg)
		case "subscriptions":
			isCarousel = false
			kbSub := v.NewKeyboard("", false)
			db.One("ID", u.ID, &vbbuser)
			if vbbuser.Subscribe9 {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscr9", unsubscribe9).SetBgColor(spColorBG))
			} else {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscr9", subscribe9).SetBgColor(spColorBG))
			}
			if vbbuser.Subscribe20 {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscr20", unsubscribe20).SetBgColor(spColorBG))
			} else {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscr20", subscribe20).SetBgColor(spColorBG))
			}
			if vbbuser.SubscribeLast {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrl", unsubscribeLast).SetBgColor(spColorBG))
			} else {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrl", subscribeLast).SetBgColor(spColorBG))
			}
			if vbbuser.SubscribeCity {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrc", unsubscribeCity).SetBgColor(spColorBG).TextSizeSmall())
			} else {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrc", subscribeCity).SetBgColor(spColorBG).TextSizeSmall())
			}
			if vbbuser.SubscribeTop {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrt", unsubscribeTop).SetBgColor(spColorBG))
			} else {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrt", subscribeTop).SetBgColor(spColorBG))
			}
			if vbbuser.SubscribeHolidays {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrh", unsubscribeHolidays).SetBgColor(spColorBG).TextSizeSmall())
			} else {
				kbSub.AddButton(v.NewTextButton(3, 1, viber.Reply, "subscrh", subscribeHolidays).SetBgColor(spColorBG).TextSizeSmall())
			}
			kbSub.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG))
			msg = v.NewTextMessage("Ваши подписки")
			msg.SetKeyboard(kbSub)
			v.SendMessage(u.ID, msg)
		case "subscr9", "conform9":
			db.One("ID", u.ID, &vbbuser)
			if txt == "subscr9" {
				msg := v.NewTextMessage("Получать дайджест за сутки - утром в 9:00")
				kb := v.NewKeyboard("", false)
				if vbbuser.Subscribe9 {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conform9", `<font color="#ffffff">Отписаться</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conform9", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG).SetSilent())
				}
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else {
				sub9 := !vbbuser.Subscribe9
				db.UpdateField(&vbbuser, "Subscribe9", sub9)
				msg := v.NewTextMessage("Спасибо")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Просмотреть подписки</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			}
		case "subscr20", "conform20":
			db.One("ID", u.ID, &vbbuser)
			if txt == "subscr20" {
				msg := v.NewTextMessage("Получать дайджест за сутки - вечером в 20:00")
				kb := v.NewKeyboard("", false)
				if vbbuser.Subscribe20 {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conform20", `<font color="#ffffff">Отписаться</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conform20", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG).SetSilent())
				}
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else {
				sub20 := !vbbuser.Subscribe20
				db.UpdateField(&vbbuser, "Subscribe20", sub20)
				msg := v.NewTextMessage("Спасибо")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Просмотреть подписки</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			}
		case "subscrl", "conforml":
			db.One("ID", u.ID, &vbbuser)
			if txt == "subscrl" {
				msg := v.NewTextMessage("Получать новости по мере их публикации.\nСообщения будут приходить часто.")
				kb := v.NewKeyboard("", false)
				if vbbuser.SubscribeLast {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conforml", `<font color="#ffffff">Отписаться</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conforml", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG).SetSilent())
				}
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else {
				subL := !vbbuser.SubscribeLast
				db.UpdateField(&vbbuser, "SubscribeLast", subL)
				msg := v.NewTextMessage("Спасибо")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Просмотреть подписки</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			}
		case "subscrc", "conformc":
			db.One("ID", u.ID, &vbbuser)
			if txt == "subscrc" {
				msg := v.NewTextMessage("Городские оповещения.")
				kb := v.NewKeyboard("", false)
				if vbbuser.SubscribeCity {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conformc", `<font color="#ffffff">Отписаться</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conformc", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG).SetSilent())
				}
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else {
				subC := !vbbuser.SubscribeCity
				db.UpdateField(&vbbuser, "SubscribeCity", subC)
				msg := v.NewTextMessage("Спасибо")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Просмотреть подписки</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			}
		case "subscrt", "conformt":
			db.One("ID", u.ID, &vbbuser)
			if txt == "subscrt" {
				msg := v.NewTextMessage("Самое популярное.")
				kb := v.NewKeyboard("", false)
				if vbbuser.SubscribeTop {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conformt", `<font color="#ffffff">Отписаться</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conformt", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG).SetSilent())
				}
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else {
				subT := !vbbuser.SubscribeTop
				db.UpdateField(&vbbuser, "SubscribeTop", subT)
				msg := v.NewTextMessage("Спасибо")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Просмотреть подписки</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			}
		case "subscrh", "conformh":
			db.One("ID", u.ID, &vbbuser)
			if txt == "subscrh" {
				msg := v.NewTextMessage("Календарь праздников.")
				kb := v.NewKeyboard("", false)
				if vbbuser.SubscribeHolidays {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conformh", `<font color="#ffffff">Отписаться</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "conformh", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG).SetSilent())
				}
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else {
				subH := !vbbuser.SubscribeHolidays
				db.UpdateField(&vbbuser, "SubscribeHolidays", subH)
				msg := v.NewTextMessage("Спасибо")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "subscriptions", `<font color="#ffffff">Просмотреть подписки</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			}
		case "alerts":
			isCarousel = true
			msgCarouselCity := v.NewRichMediaMessage(6, 7, spColorBG)
			var city News
			numPage := 0
			db.One("ID", u.ID, &vbbuser)
			var msgText string
			kb := v.NewKeyboard("", false)
			urlCity := botConfig.QueryCityDisp
			city, err := NewsQuery(urlCity, numPage)
			if err != nil {
				log.Println(err)
			}
			v.SendTextMessage(u.ID, "Городские оповещения")
			for _, cityItem := range city.Nodes {
				srcDate := cityItem.Node.NodeDate
				msgCarouselCity.AddButton(v.NewTextButton(6, 2, viber.OpenURL, cityItem.Node.NodePath, strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[1]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[0]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[2]+strings.SplitAfter(srcDate, " ")[3]+"\n"+cityItem.Node.NodeTitle))
				msgCarouselCity.AddButton(v.NewImageButton(6, 4, viber.OpenURL, cityItem.Node.NodePath, cityItem.Node.NodeCover["src"]))
				msgCarouselCity.AddButton(v.NewTextButton(6, 1, viber.OpenURL, cityItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			urlCity = botConfig.QueryCityAfisha
			city, err = NewsQuery(urlCity, numPage)
			for _, cityItem := range city.Nodes {
				srcDate := cityItem.Node.NodeDate
				msgCarouselCity.AddButton(v.NewTextButton(6, 2, viber.OpenURL, cityItem.Node.NodePath, strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[1]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[0]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[2]+strings.SplitAfter(srcDate, " ")[3]+"\n"+cityItem.Node.NodeTitle))
				msgCarouselCity.AddButton(v.NewImageButton(6, 4, viber.OpenURL, cityItem.Node.NodePath, cityItem.Node.NodeCover["src"]))
				msgCarouselCity.AddButton(v.NewTextButton(6, 1, viber.OpenURL, cityItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			v.SendMessage(u.ID, msgCarouselCity)
			if !vbbuser.SubscribeCity {
				msgText = "Оформив подписку на городские оповещения, Вы будете получать предупреждения городских служб, анонсы мероприятий в Бельцах и т.д."
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "conformc", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG))
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "menu", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG))
			} else {
				msgText = "Спасибо"
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG))
			}
			msg := v.NewTextMessage(msgText)
			msg.SetKeyboard(kb)
			v.SendMessage(u.ID, msg)
		case "top":
			isCarousel = true
			msgCarouselView := v.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselComment := v.NewRichMediaMessage(6, 7, spColorBG)
			var top News
			db.One("ID", u.ID, &vbbuser)
			var msgText string
			kb := v.NewKeyboard("", false)
			urlTop := botConfig.QueryTopViews
			top, err := NewsQuery(urlTop, -1)
			if err != nil {
				log.Println(err)
			}
			v.SendTextMessage(u.ID, "Самые читаемые за последние семь дней")
			for _, topItem := range top.Nodes {
				msgCarouselView.AddButton(v.NewTextButton(6, 2, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeDate+"\n"+topItem.Node.NodeTitle))
				msgCarouselView.AddButton(v.NewImageButton(6, 4, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeCover["src"]))
				msgCarouselView.AddButton(v.NewTextButton(6, 1, viber.OpenURL, topItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			v.SendMessage(u.ID, msgCarouselView)
			urlTop = botConfig.QueryTopComments
			top, err = NewsQuery(urlTop, -1)
			if err != nil {
				log.Println(err)
			}
			v.SendTextMessage(u.ID, "Самые комментируемые за последние семь дней")
			for _, topItem := range top.Nodes {
				msgCarouselComment.AddButton(v.NewTextButton(6, 2, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeDate+"\n"+topItem.Node.NodeTitle))
				msgCarouselComment.AddButton(v.NewImageButton(6, 4, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeCover["src"]))
				msgCarouselComment.AddButton(v.NewTextButton(6, 1, viber.OpenURL, topItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			v.SendMessage(u.ID, msgCarouselComment)
			if !vbbuser.SubscribeTop {
				msgText = `Хотите подписаться на самое популярное в "СП"? мы будем присылать вам такие подборки каждое воскресенье в 10:00.`
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "conformt", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG))
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "menu", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG))
			} else {
				msgText = "Спасибо"
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG))
			}
			msg := v.NewTextMessage(msgText)
			msg.SetKeyboard(kb)
			v.SendMessage(u.ID, msg)
		case "news", "newsprev", "newsnext":
			isCarousel = true
			msgCarouselLast240 := v.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselLast241 := v.NewRichMediaMessage(6, 7, spColorBG)
			msgNavig := v.NewRichMediaMessage(6, 3, "#ffffff")
			if txt == "news" {
				v.SendTextMessage(u.ID, "Последние новости")
				Page = 0
				msgNavig.AddButton(v.NewTextButton(6, 2, viber.Reply, "newsnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
			} else if txt == "newsnext" {
				Page++
				if Page == 800 {
					msgNavig.AddButton(v.NewTextButton(6, 2, viber.Reply, "newsprev", `<font color="#ffffff">Назад</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "newsprev", `<font color="#ffffff">Назад</font>`).SetBgColor(spColorBG).SetSilent())
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "newsnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
				}
			} else {
				Page--
				if Page != 0 {
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "newsprev", `<font color="#ffffff">Назад</font>`).SetBgColor(spColorBG).SetSilent())
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "newsnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					msgNavig.AddButton(v.NewTextButton(6, 2, viber.Reply, "newsnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
				}
			}
			var last24 News
			urlLast24 := botConfig.QueryNews24H
			last24, err := NewsQuery(urlLast24, Page)
			if err != nil {
				log.Println(err)
			}
			for i, last24Item := range last24.Nodes {
				if i < 5 {
					msgCarouselLast240.AddButton(v.NewTextButton(6, 2, viber.OpenURL, last24Item.Node.NodePath, last24Item.Node.NodeDate+"\n"+last24Item.Node.NodeTitle))
					msgCarouselLast240.AddButton(v.NewImageButton(6, 4, viber.OpenURL, last24Item.Node.NodePath, last24Item.Node.NodeCover["src"]))
					msgCarouselLast240.AddButton(v.NewTextButton(6, 1, viber.OpenURL, last24Item.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
				} else {
					msgCarouselLast241.AddButton(v.NewTextButton(6, 2, viber.OpenURL, last24Item.Node.NodePath, last24Item.Node.NodeDate+"\n"+last24Item.Node.NodeTitle))
					msgCarouselLast241.AddButton(v.NewImageButton(6, 4, viber.OpenURL, last24Item.Node.NodePath, last24Item.Node.NodeCover["src"]))
					msgCarouselLast241.AddButton(v.NewTextButton(6, 1, viber.OpenURL, last24Item.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
				}
			}
			v.SendMessage(u.ID, msgCarouselLast240)
			v.SendMessage(u.ID, msgCarouselLast241)
			msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			v.SendMessage(u.ID, msgNavig)
		case "search":
			isSearch = true
			v.SendTextMessage(u.ID, "Введите слово или фразу для поиска")
		case "searchbegin", "searchprev", "searchnext":
			isCarousel = true
			msgCarouselSearch := v.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselSearch1 := v.NewRichMediaMessage(6, 7, spColorBG)
			msgNavig := v.NewRichMediaMessage(6, 3, "#ffffff")
			notFound := false
			if txt == "searchbegin" {
				var search News
				Page = 0
				search, err := SearchQuery(searchString, Page)
				if err != nil {
					log.Println(err)
				}
				if len(search.Nodes) == 0 {
					notFound = true
					msg = v.NewTextMessage("По вашему запросу ничего не найдено")
					msg.SetKeyboard(kbMain)
					v.SendMessage(u.ID, msg)
				} else {
					for i, searchItem := range search.Nodes {
						if i < 5 {
							msgCarouselSearch.AddButton(v.NewTextButton(6, 2, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeDate+"\n"+searchItem.Node.NodeTitle))
							msgCarouselSearch.AddButton(v.NewImageButton(6, 4, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeCover["src"]))
							msgCarouselSearch.AddButton(v.NewTextButton(6, 1, viber.OpenURL, searchItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
						} else {
							msgCarouselSearch1.AddButton(v.NewTextButton(6, 2, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeDate+"\n"+searchItem.Node.NodeTitle))
							msgCarouselSearch1.AddButton(v.NewImageButton(6, 4, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeCover["src"]))
							msgCarouselSearch1.AddButton(v.NewTextButton(6, 1, viber.OpenURL, searchItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
						}
					}
					v.SendMessage(u.ID, msgCarouselSearch)
					v.SendMessage(u.ID, msgCarouselSearch1)
				}
				msgNavig.AddButton(v.NewTextButton(6, 2, viber.Reply, "searchnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
			} else if txt == "searchnext" {
				Page++
				search, err := SearchQuery(searchString, Page)
				if err != nil {
					log.Println(err)
				}
				for i, searchItem := range search.Nodes {
					if i < 5 {
						msgCarouselSearch.AddButton(v.NewTextButton(6, 2, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeDate+"\n"+searchItem.Node.NodeTitle))
						msgCarouselSearch.AddButton(v.NewImageButton(6, 4, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeCover["src"]))
						msgCarouselSearch.AddButton(v.NewTextButton(6, 1, viber.OpenURL, searchItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
					} else {
						msgCarouselSearch1.AddButton(v.NewTextButton(6, 2, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeDate+"\n"+searchItem.Node.NodeTitle))
						msgCarouselSearch1.AddButton(v.NewImageButton(6, 4, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeCover["src"]))
						msgCarouselSearch1.AddButton(v.NewTextButton(6, 1, viber.OpenURL, searchItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
					}
				}
				v.SendMessage(u.ID, msgCarouselSearch)
				v.SendMessage(u.ID, msgCarouselSearch1)
				if len(search.Nodes) == 0 || len(search.Nodes) < 10 {
					msgNavig.AddButton(v.NewTextButton(6, 2, viber.Reply, "searchprev", `<font color="#ffffff">Назад</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "searchprev", `<font color="#ffffff">Назад</font>`).SetBgColor(spColorBG).SetSilent())
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "searchnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
				}
			} else {
				Page--
				if Page < 0 {
					Page = 0
				}
				search, err := SearchQuery(searchString, Page)
				if err != nil {
					log.Println(err)
				}
				for i, searchItem := range search.Nodes {
					if i < 5 {
						msgCarouselSearch.AddButton(v.NewTextButton(6, 2, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeDate+"\n"+searchItem.Node.NodeTitle))
						msgCarouselSearch.AddButton(v.NewImageButton(6, 4, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeCover["src"]))
						msgCarouselSearch.AddButton(v.NewTextButton(6, 1, viber.OpenURL, searchItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
					} else {
						msgCarouselSearch1.AddButton(v.NewTextButton(6, 2, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeDate+"\n"+searchItem.Node.NodeTitle))
						msgCarouselSearch1.AddButton(v.NewImageButton(6, 4, viber.OpenURL, searchItem.Node.NodePath, searchItem.Node.NodeCover["src"]))
						msgCarouselSearch1.AddButton(v.NewTextButton(6, 1, viber.OpenURL, searchItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
					}
				}
				v.SendMessage(u.ID, msgCarouselSearch)
				v.SendMessage(u.ID, msgCarouselSearch1)
				if Page != 0 {
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "searchprev", `<font color="#ffffff">Назад</font>`).SetBgColor(spColorBG).SetSilent())
					msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "searchnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
				} else {
					msgNavig.AddButton(v.NewTextButton(6, 2, viber.Reply, "searchnext", `<font color="#ffffff">Вперед</font>`).SetBgColor(spColorBG).SetSilent())
				}
			}
			if !notFound {
				msgNavig.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				v.SendMessage(u.ID, msgNavig)
			}
		case "feedback", "sendfeedback":
			if txt == "feedback" {
				isCarousel = false
				isFeedback = true
				attachmentCount = 5
				attachmentURLs = nil
				fileName = nil
				contentType = nil
				emailSubject = "Viber\n"
				emailSubject += "Сообщение от: ID:" + u.ID + " Username:" + u.Name + "\n"
				emailSubject += "Дата:" + t.String()
				v.SendTextMessage(u.ID, feedbackText)
			} else {
				go func(emailSubject string, emailBody string, attachmentURLs []string, fileName []string, contentType []string) {
					err := SendFeedback(emailSubject, emailBody, attachmentURLs, fileName, contentType)
					if err != nil {
						log.Printf("Send feedback err: %#+v", err)
					}
				}(emailSubject, emailBody, attachmentURLs, fileName, contentType)
				isFeedback = false
				msg := v.NewTextMessage("Спасибо")
				msg.SetKeyboard(kbMain)
				v.SendMessage(u.ID, msg)
			}
		case "holidays":
			isCarousel = false
			var msgText string
			db.One("ID", u.ID, &vbbuser)
			kb := v.NewKeyboard("", false)
			if NoWork {
				msgText = "Извините. Пока не доступно."
			} else {
				msgText = "Молдавские, международные и религиозные праздники из нашего календаря	\"Существенный Повод\" на ближайшую неделю:\n\n"
				for _, hd := range HolidayList {
					if (hd.Date.Unix() >= time.Now().AddDate(0, 0, -1).Unix()) && (hd.Date.Unix() <= time.Now().AddDate(0, 0, 7).Unix()) {
						msgText += "*" + hd.Day + " " + hd.Month + "*" + "\n" + hd.Holiday + "\n\n"
					}
				}
			}
			if !vbbuser.SubscribeHolidays {
				msgText += "Предлагаем вам подписаться на рассылку праздников. Мы будем присылать вам даты на неделю каждый понедельник в 10:00"
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "conformh", `<font color="#ffffff">Подписаться</font>`).SetBgColor(spColorBG))
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "menu", `<font color="#ffffff">Нет, спасибо</font>`).SetBgColor(spColorBG))
			} else {
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG))
			}
			msg := v.NewTextMessage(msgText)
			msg.SetKeyboard(kb)
			v.SendMessage(u.ID, msg)
		case "games", "games10", "games1rand":
			isCarousel = true
			msgCarouselGames := v.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselGames1 := v.NewRichMediaMessage(6, 7, spColorBG)
			if txt == "games" {
				msg := v.NewTextMessage("Выбирите игру")
				kb := v.NewKeyboard("#ffffff", false)
				kb.AddButton(v.NewTextButton(3, 2, viber.Reply, "games10", `<font color="#ffffff">Последние 10</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(3, 2, viber.Reply, "games1rand", `<font color="#ffffff">Случайная</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
			} else if txt == "games10" {
				var games News
				urlGames := botConfig.QueryGames
				games, err := NewsQuery(urlGames, 0)
				if err != nil {
					log.Println(err)
				}
				for i, gamesItem := range games.Nodes {
					if i < 5 {
						msgCarouselGames.AddButton(v.NewTextButton(6, 2, viber.OpenURL, gamesItem.Node.NodePath, gamesItem.Node.NodeDate+"\n"+gamesItem.Node.NodeTitle))
						msgCarouselGames.AddButton(v.NewImageButton(6, 4, viber.OpenURL, gamesItem.Node.NodePath, gamesItem.Node.NodeCover["src"]))
						msgCarouselGames.AddButton(v.NewTextButton(6, 1, viber.OpenURL, gamesItem.Node.NodePath, `<font color="#ffffff">Играть...</font>`).SetBgColor(spColorBG))
					} else {
						msgCarouselGames1.AddButton(v.NewTextButton(6, 2, viber.OpenURL, gamesItem.Node.NodePath, gamesItem.Node.NodeDate+"\n"+gamesItem.Node.NodeTitle))
						msgCarouselGames1.AddButton(v.NewImageButton(6, 4, viber.OpenURL, gamesItem.Node.NodePath, gamesItem.Node.NodeCover["src"]))
						msgCarouselGames1.AddButton(v.NewTextButton(6, 1, viber.OpenURL, gamesItem.Node.NodePath, `<font color="#ffffff">Играть...</font>`).SetBgColor(spColorBG))
					}
				}
				v.SendMessage(u.ID, msgCarouselGames)
				v.SendMessage(u.ID, msgCarouselGames1)
			} else {
				var games News
				urlGames := botConfig.QueryGames
				games, err := NewsQuery(urlGames, 0)
				if err != nil {
					log.Println(err)
				}
				rand.Seed(time.Now().UTC().UnixNano())
				choice := rand.Intn(10)
				gamesItem := games.Nodes[choice]
				msgCarouselGames.AddButton(v.NewTextButton(6, 2, viber.OpenURL, gamesItem.Node.NodePath, gamesItem.Node.NodeDate+"\n"+gamesItem.Node.NodeTitle))
				msgCarouselGames.AddButton(v.NewImageButton(6, 4, viber.OpenURL, gamesItem.Node.NodePath, gamesItem.Node.NodeCover["src"]))
				msgCarouselGames.AddButton(v.NewTextButton(6, 1, viber.OpenURL, gamesItem.Node.NodePath, `<font color="#ffffff">Играть...</font>`).SetBgColor(spColorBG))
				v.SendMessage(u.ID, msgCarouselGames)
			}
		case "donate":
			isCarousel = true
			msg = v.NewTextMessage(`Мы предлагаем поддержать независимую команду "СП", подписавшись на нашу газету (печатная или PDF-версии) или сделав финансовый вклад в нашу работу.`)
			kb := v.NewKeyboard("#ffffff", false)
			kb.AddButton(v.NewTextButton(3, 2, viber.OpenURL, "http://esp.md/content/podpiska-na-sp", `<font color="#ffffff">Подписаться на газету "СП"</font>`).SetBgColor(spColorBG).SetSilent())
			kb.AddButton(v.NewTextButton(3, 2, viber.OpenURL, "http://esp.md/donate", `<font color="#ffffff">Поддержать "СП" материально</font>`).SetBgColor(spColorBG).SetSilent())
			kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			msg.SetKeyboard(kb)
			v.SendMessage(u.ID, msg)
		case "menu":
			isFeedback = false
			isSearch = false
			isCarousel = false
			msg = v.NewTextMessage("Выбирете команду")
			msg.SetKeyboard(kbMain)
			v.SendMessage(u.ID, msg)
		case "hi", "hello", "хай", "привет", "рш", "руддщ":
			if !isFeedback {
				isFeedback = false
				isSearch = false
				isCarousel = false
				msg = v.NewTextMessage("Привет!\nВыбирете команду")
				msg.SetKeyboard(kbMain)
				v.SendMessage(u.ID, msg)
				break
			}
			fallthrough
		default:
			if isFeedback {
				emailBody = m.(*viber.TextMessage).Text
				msg := v.NewTextMessage("Вы можете прикрепить до " + strconv.Itoa(attachmentCount) + " файлов к сообщению")
				kb := v.NewKeyboard("", false)
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "sendfeedback", `<font color="#ffffff">Отправить</font>`).SetBgColor(spColorBG))
				kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "menu", `<font color="#ffffff">Отменить</font>`).SetBgColor(spColorBG))
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
				break
			}
			if isSearch {
				searchString = m.(*viber.TextMessage).Text
				msg = v.NewTextMessage("Начинаем поиск")
				kb := v.NewKeyboard("#ffffff", false)
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "searchbegin", `<font color="#ffffff">Искать</font>`).SetBgColor(spColorBG).SetSilent())
				kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
				msg.SetKeyboard(kb)
				v.SendMessage(u.ID, msg)
				isSearch = false
				break
			}
			if !isCarousel {
				msg = v.NewTextMessage(noCmdText)
				msg.SetKeyboard(kbMain)
				v.SendMessage(u.ID, msg)
			} else {
				isCarousel = false
				msg = v.NewTextMessage("Главное меню")
				msg.SetKeyboard(kbMain)
				v.SendMessage(u.ID, msg)
			}
		}
	case *viber.URLMessage:
		url := m.(*viber.URLMessage).Media
		v.SendTextMessage(u.ID, "You send me this URL:"+url)
	case *viber.PictureMessage:
		msg := v.NewTextMessage("")
		kb := v.NewKeyboard("", false)
		kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "sendfeedback", `<font color="#ffffff">Отправить</font>`).SetBgColor(spColorBG))
		kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "menu", `<font color="#ffffff">Отменить</font>`).SetBgColor(spColorBG))
		msg.SetKeyboard(kb)
		if isFeedback && attachmentCount > 0 {
			attachmentURLs = append(attachmentURLs, m.(*viber.PictureMessage).Media)
			fileName = append(fileName, "File"+strconv.Itoa(6-attachmentCount)+".jpg")
			contentType = append(contentType, "image/jpeg")
			attachmentCount--
			msg.Text = "Вы можете прикрепить еще " + strconv.Itoa(attachmentCount) + " файла/ов"
		} else {
			msg.Text = "Вы исчерпали лимит вложений"
		}
		v.SendMessage(u.ID, msg)
	case *viber.VideoMessage:
		msg := v.NewTextMessage("")
		kb := v.NewKeyboard("", false)
		kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "sendfeedback", `<font color="#ffffff">Отправить</font>`).SetBgColor(spColorBG))
		kb.AddButton(v.NewTextButton(3, 1, viber.Reply, "menu", `<font color="#ffffff">Отменить</font>`).SetBgColor(spColorBG))
		msg.SetKeyboard(kb)
		if isFeedback && attachmentCount > 0 {
			attachmentURLs = append(attachmentURLs, m.(*viber.VideoMessage).Media)
			fileName = append(fileName, "File"+strconv.Itoa(6-attachmentCount)+".mp4")
			contentType = append(contentType, "video/mp4")
			attachmentCount--
			msg.Text = "Вы можете прикрепить еще " + strconv.Itoa(attachmentCount) + " файла/ов"
		} else {
			msg.Text = "Вы исчерпали лимит вложений"
		}
		v.SendMessage(u.ID, msg)
	}
	err = db.One("ID", u.ID, &vbbuser)
	if err == nil {
		db.UpdateField(&vbbuser, "LastDate", t)
	} else {
		vbbuser.ID = u.ID
		vbbuser.Username = u.Name
		vbbuser.LastDate = t
		vbbuser.Subscribe9 = false
		vbbuser.Subscribe20 = false
		vbbuser.SubscribeLast = false
		vbbuser.SubscribeCity = false
		vbbuser.SubscribeTop = false
		vbbuser.SubscribeHolidays = false
		db.Save(&vbbuser)
	}
}

//msgConversationStarted call when user opens conversation
func msgConversationStarted(v *viber.Viber, u viber.User, conversationType string, context string, subscribed bool, token uint64, t time.Time) viber.Message {
	welcomeMessage := `Здравствуйте! Подключайтесь к новостному боту "СП" - умному ассистенту, который поможет вам получать полезную и важную информацию в телефоне удобным для вас образом.
	Чтобы начать работу с ботом нажмите кнопку "Начать общение" или отправьте боту любое сообщение,например поздоровайтесь с ботом (Hi, Hello, Привет).`
	kb := v.NewKeyboard("", false)
	kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "start", `<font color="#ffffff">Начать общение</font>`).SetBgColor(spColorBG))
	msg := v.NewTextMessage(welcomeMessage)
	msg.SetKeyboard(kb)
	return msg
}

//msgSubscribed call when user subscribe on bot
func msgSubscribed(v *viber.Viber, u viber.User, token uint64, t time.Time) {
	subscribeMessage := v.NewTextMessage(`Здравствуйте! Подключайтесь к новостному боту "СП" - умному ассистенту, который поможет вам получать полезную и важную информацию в телефоне удобным для вас образом.
	Чтобы начать работу с ботом нажмите кнопку "Начать общение" или отправьте боту любое сообщение,например поздоровайтесь с ботом (Hi, Hello, Привет).`)
	kb := v.NewKeyboard("", false)
	kb.AddButton(v.NewTextButton(6, 1, viber.Reply, "start", `<font color="#ffffff">Начать общение</font>`).SetBgColor(spColorBG).SetSilent())
	subscribeMessage.SetKeyboard(kb)
	v.SendMessage(u.ID, subscribeMessage)
}
