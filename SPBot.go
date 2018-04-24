package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/SlyMarbo/rss"

	"github.com/Syfaro/telegram-bot-api"
	"github.com/asdine/storm"
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

func main() {
	config, err := LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	// Bolt
	db, err := storm.Open("user.db")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
	// Telegram users from db Bucket tgUsers
	var tgbUser TgUser
	db.Init(&tgbUser)

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
	startMsgText := `Добро пожаловать! Предлагаем Вам подписаться на новости на сайте "СП". Вы сможете настроить рассылку так, как Вам удобно.`
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
	startMsgEndText := `Спасибо за Ваш выбор! Вы можете отписаться от нашей рассылки в любой момент в меню /subscriptions`
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
	// Test RSS
	feed, err := rss.Fetch("http://esp.md/feed/rss")
	// fmt.Println(feed)
	// Cron for subscriptions
	c := cron.New()
	// c.AddFunc("0 0/5 * * * *", func() {
	// tg40Msg := tgbotapi.NewMessage(474165300, startMsgText)
	// tg40Msg.ParseMode = "Markdown"
	// tgBot.Send(tg40Msg)
	// feed.Update()
	// fmt.Println(feed)
	// })
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
				case "subscribestart":
					tgCbMsg.Text = `Выберите подписку:
					Утром - получать дайджест за сутки утром - в 9:00
					Вечером - получать дайджест за сутки вечером - в 20:00
					Последние новости - получать новости сразу по мере их публикации _(сообщения будут приходить часто)_`
					buttonSubscribe9 := tgbotapi.NewInlineKeyboardButtonData("Утром", "subscribe9start")
					buttonSubscribe20 := tgbotapi.NewInlineKeyboardButtonData("Вечером", "subscribe20start")
					buttonSubscribeLast := tgbotapi.NewInlineKeyboardButtonData("Последние новости", "subscribelaststart")
					var row []tgbotapi.InlineKeyboardButton
					var row1 []tgbotapi.InlineKeyboardButton
					var row2 []tgbotapi.InlineKeyboardButton
					row = append(row, buttonSubscribe9)
					row1 = append(row1, buttonSubscribe20)
					row2 = append(row2, buttonSubscribeLast)
					keyboard := tgbotapi.NewInlineKeyboardMarkup(row, row1, row2)
					tgCbMsg.ReplyMarkup = keyboard
				case "subscribe9start":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "Subscribe9", true)
					tgCbMsg.Text = startMsgEndText
				case "subscribe20start":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "Subscribe20", true)
					tgCbMsg.Text = startMsgEndText
				case "subscribelaststart":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "SubscribeLast", true)
					tgCbMsg.Text = startMsgEndText
				case "subscribe9":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSub9 := !tgbUser.Subscribe9
					db.UpdateField(&tgbUser, "Subscribe9", changeSub9)
					tgCbMsg.Text = startMsgEndText
				case "subscribe20":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSub20 := !tgbUser.Subscribe20
					db.UpdateField(&tgbUser, "Subscribe20", changeSub20)
					tgCbMsg.Text = startMsgEndText
				case "subscribelast":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubLast := !tgbUser.SubscribeLast
					db.UpdateField(&tgbUser, "SubscribeLast", changeSubLast)
					tgCbMsg.Text = startMsgEndText
				case "subscribetop":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubTop := !tgbUser.SubscribeTop
					db.UpdateField(&tgbUser, "SubscribeTop", changeSubTop)
					tgCbMsg.Text = startMsgEndText
				case "subscribecity":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubCity := !tgbUser.SubscribeCity
					db.UpdateField(&tgbUser, "SubscribeCity", changeSubCity)
					tgCbMsg.Text = startMsgEndText
				case "subscribeholidays":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubHolidays := !tgbUser.SubscribeHolidays
					db.UpdateField(&tgbUser, "SubscribeHolidays", changeSubHolidays)
					tgCbMsg.Text = startMsgEndText
				}
				err = db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
				if err == nil {
					db.UpdateField(&tgbUser, "LastDate", tgUpdate.CallbackQuery.Message.Date)
				} else {
					tgbUser.LastDate = tgUpdate.CallbackQuery.Message.Date
					tgbUser.ChatID = tgUpdate.CallbackQuery.Message.Chat.ID
					tgbUser.Username = tgUpdate.CallbackQuery.Message.Chat.UserName
					tgbUser.FirstName = tgUpdate.CallbackQuery.Message.Chat.FirstName
					tgbUser.LastName = tgUpdate.CallbackQuery.Message.Chat.LastName
					tgbUser.Subscribe9 = false
					tgbUser.Subscribe20 = false
					tgbUser.SubscribeLast = false
					tgbUser.SubscribeTop = false
					tgbUser.SubscribeCity = false
					tgbUser.SubscribeHolidays = false
					tgbUser.RssLastID = 0
					db.Save(&tgbUser)
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
				buttonSubscribe := tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribestart")
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Нет, спасибо", "help")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSubscribe, buttonHelp))
				tgMsg.ReplyMarkup = keyboard
			case "subscriptions":
				bt9 := "Утром"
				bt20 := "Вечером"
				btL := "Последние новости"
				btT := "Самое популярное"
				btC := "Городские уведомления"
				btH := "Календарь праздников"
				db.One("ChatID", tgUpdate.Message.Chat.ID, &tgbUser)
				if tgbUser.Subscribe9 {
					bt9 = "\u2705" + bt9
				}
				if tgbUser.Subscribe20 {
					bt20 = "\u2705" + bt20
				}
				if tgbUser.SubscribeLast {
					btL = "\u2705" + btL
				}
				if tgbUser.SubscribeTop {
					btT = "\u2705" + btT
				}
				if tgbUser.SubscribeCity {
					btC = "\u2705" + btC
				}
				if tgbUser.SubscribeHolidays {
					btH = "\u2705" + btH
				}
				tgMsg.Text = `Управление подписками:
					*Утром* - получать дайджест за сутки утром - в 9:00
					*Вечером* - получать дайджест за сутки вечером - в 20:00
					*Последние новости* - получать новости сразу по мере их 
					публикации _(сообщения будут приходить часто)_
					*Самое популярное* - самые читаемые и комментируемые материалы за 7 дней
					*Городские уведомления* - предупреждения городских служб, анонсы мероприятий в Бельцах и т.п.
					*Календарь праздников* - молдавские, международные и религиозные праздники на ближайшую неделю
					
						Для изменения состояния подписки нажмите на 
					соответствующую кнопку
					_Символ ✔ стоит около рассылок к которым Вы подписаны_`
				buttonSubscribe9 := tgbotapi.NewInlineKeyboardButtonData(bt9, "subscribe9")
				buttonSubscribe20 := tgbotapi.NewInlineKeyboardButtonData(bt20, "subscribe20")
				buttonSubscribeLast := tgbotapi.NewInlineKeyboardButtonData(btL, "subscribelast")
				buttonSubscribeTop := tgbotapi.NewInlineKeyboardButtonData(btT, "subscribetop")
				buttonSubscribeCity := tgbotapi.NewInlineKeyboardButtonData(btC, "subscribecity")
				buttonSubscribeHolidays := tgbotapi.NewInlineKeyboardButtonData(btH, "subscribeholidays")
				var row []tgbotapi.InlineKeyboardButton
				var row1 []tgbotapi.InlineKeyboardButton
				var row2 []tgbotapi.InlineKeyboardButton
				row = append(row, buttonSubscribe9)
				row = append(row, buttonSubscribe20)
				row1 = append(row1, buttonSubscribeLast)
				row1 = append(row1, buttonSubscribeTop)
				row2 = append(row2, buttonSubscribeCity)
				row2 = append(row2, buttonSubscribeHolidays)
				keyboard := tgbotapi.NewInlineKeyboardMarkup(row, row1, row2)
				tgMsg.ReplyMarkup = keyboard
			case "beltsy":
				tgMsg.Text = stubMsgText
			case "top":
				tgMsg.Text = stubMsgText
			case "news":
				feed.Update()
				for _, newsItem := range feed.Items {
					tgMsg.Text = "[" + newsItem.Title + "\n" + newsItem.Date.Format("02-01-2006 15:04") + "]" + "(" + newsItem.Link + ")"
					tgBot.Send(tgMsg)
				}
				continue
			case "search":
				tgMsg.Text = stubMsgText
			case "feedback":
				tgMsg.Text = stubMsgText
			case "holidays":
				tgMsg.Text = strconv.Itoa(int(tgUpdate.Message.Chat.ID)) + "\u2714" + tgUpdate.Message.Chat.FirstName + time.Unix(int64(tgUpdate.Message.Date), 0).String()
			case "games":
				tgMsg.Text = stubMsgText
			case "donate":
				tgMsg.Text = `Мы предлагаем поддержать независимую комманду "СП", подписавшись на нашу газету (печатная или PDF-версии) или сделав финансовый вклад в нашу работу.`
				buttonSubscribe := tgbotapi.NewInlineKeyboardButtonURL("Подписаться на газету \"СП\"", "http://esp.md/content/podpiska-na-sp")
				buttonDonate := tgbotapi.NewInlineKeyboardButtonURL("Поддержать \"СП\" материально", "http://esp.md/donate")
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Вернуться в главное меню", "help")
				var row []tgbotapi.InlineKeyboardButton
				var row1 []tgbotapi.InlineKeyboardButton
				var row2 []tgbotapi.InlineKeyboardButton
				row = append(row, buttonSubscribe)
				row1 = append(row1, buttonDonate)
				row2 = append(row2, buttonHelp)
				keyboard := tgbotapi.NewInlineKeyboardMarkup(row, row1, row2)
				tgMsg.ReplyMarkup = keyboard
			default:
				toOriginal = true
				tgMsg.Text = noCmdText
			}

			if toOriginal {
				tgMsg.ReplyToMessageID = tgUpdate.Message.MessageID
			}
			err = db.One("ChatID", tgUpdate.Message.Chat.ID, &tgbUser)
			if err == nil {
				db.UpdateField(&tgbUser, "LastDate", tgUpdate.Message.Date)
			} else {
				tgbUser.LastDate = tgUpdate.Message.Date
				tgbUser.ChatID = tgUpdate.Message.Chat.ID
				tgbUser.Username = tgUpdate.Message.Chat.UserName
				tgbUser.FirstName = tgUpdate.Message.Chat.FirstName
				tgbUser.LastName = tgUpdate.Message.Chat.LastName
				tgbUser.Subscribe9 = false
				tgbUser.Subscribe20 = false
				tgbUser.SubscribeLast = false
				tgbUser.SubscribeTop = false
				tgbUser.SubscribeCity = false
				tgbUser.SubscribeHolidays = false
				tgbUser.RssLastID = 0
				db.Save(&tgbUser)
			}
			tgBot.Send(tgMsg)
			// Default may cause high CPU load
			// default:
		}
	}
}
