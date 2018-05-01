package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/SlyMarbo/rss"

	"github.com/Syfaro/telegram-bot-api"
	"github.com/asdine/storm"
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron"
)

func main() {
	config, err := LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	// Load holidays if error send message not released
	noWork := false
	holidays, err := LoadHolidays(config.FileHolidays)
	if err != nil {
		log.Println(err)
		noWork = true
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
	}
	defer watcher.Close()
	err = watcher.Add(config.FileHolidays)
	if err != nil {
		log.Println(err)
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
	// RSS
	feed, err := rss.Fetch("http://esp.md/feed/rss")
	var countFeed int
	countView := 5
	// Cron for subscriptions
	c := cron.New()
	c.AddFunc("0 0/15 * * * *", func() {
		// tg40Msg := tgbotapi.NewMessage(474165300, startMsgText)
		// tg40Msg.ParseMode = "Markdown"
		// tgBot.Send(tg40Msg)
		// feed.Update()
		fmt.Println(time.Now(), "Tik-Tak")
	})
	c.AddFunc("@hourly", func() {
		// tg1hMsg := tgbotapi.NewMessage(474165300, "Ku-Ku")
		// tg1hMsg.ParseMode = "Markdown"
		// tgBot.Send(tg1hMsg)
		fmt.Println(time.Now(), "Tik-Tak 1 Hour")
	})
	c.Start()

	// Get updates from channels
	for {

		select {
		// Watch holidays.txt and update Holidays
		case event := <-watcher.Events:
			log.Println("event:", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file:", event.Name)
			}
			holidays, err = LoadHolidays(config.FileHolidays)
			if err != nil {
				log.Println(err)
				noWork = true
			} else {
				noWork = false
			}
		case errEv := <-watcher.Errors:
			log.Println("error:", errEv)
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
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
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
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
				case "subscribe9start":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "Subscribe9", true)
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "subscribe20start":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "Subscribe20", true)
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "subscribelaststart":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "SubscribeLast", true)
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "subscribe9":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSub9 := !tgbUser.Subscribe9
					db.UpdateField(&tgbUser, "Subscribe9", changeSub9)
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					newmessage := SubButtons(&tgUpdate, &tgbUser)
					tgBot.Send(newmessage)
				case "subscribe20":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSub20 := !tgbUser.Subscribe20
					db.UpdateField(&tgbUser, "Subscribe20", changeSub20)
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					newmessage := SubButtons(&tgUpdate, &tgbUser)
					tgBot.Send(newmessage)
				case "subscribelast":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubLast := !tgbUser.SubscribeLast
					if !changeSubLast {
						db.UpdateField(&tgbUser, "RssLastID", 0)
					}
					db.UpdateField(&tgbUser, "SubscribeLast", changeSubLast)
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					newmessage := SubButtons(&tgUpdate, &tgbUser)
					tgBot.Send(newmessage)
				case "subscribetop":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubTop := !tgbUser.SubscribeTop
					db.UpdateField(&tgbUser, "SubscribeTop", changeSubTop)
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					newmessage := SubButtons(&tgUpdate, &tgbUser)
					tgBot.Send(newmessage)
				case "subscribecity":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubCity := !tgbUser.SubscribeCity
					db.UpdateField(&tgbUser, "SubscribeCity", changeSubCity)
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					newmessage := SubButtons(&tgUpdate, &tgbUser)
					tgBot.Send(newmessage)
				case "subscribeholidays":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					changeSubHolidays := !tgbUser.SubscribeHolidays
					db.UpdateField(&tgbUser, "SubscribeHolidays", changeSubHolidays)
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					newmessage := SubButtons(&tgUpdate, &tgbUser)
					tgBot.Send(newmessage)
				case "subscribefinish":
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "subscribehd":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "SubscribeHolidays", true)
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "subscribetp":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "SubscribeTop", true)
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "subscribec":
					db.One("ChatID", tgUpdate.CallbackQuery.Message.Chat.ID, &tgbUser)
					db.UpdateField(&tgbUser, "SubscribeCity", true)
					tgBot.DeleteMessage(tgbotapi.DeleteMessageConfig{ChatID: tgUpdate.CallbackQuery.Message.Chat.ID, MessageID: tgUpdate.CallbackQuery.Message.MessageID})
					tgCbMsg.Text = startMsgEndText
				case "next5":
					buttonNext5 := tgbotapi.NewInlineKeyboardButtonData("Следующие "+strconv.Itoa(countView)+" новостей", "next5")
					keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNext5))
					for count := countFeed + 1; count < len(feed.Items); count++ {
						if count == countFeed+countView {
							countFeed = count
							if count != len(feed.Items)-1 {
								tgCbMsg.ReplyMarkup = keyboard
							}
							tgCbMsg.Text = "[" + feed.Items[count].Title + "\n" + feed.Items[count].Date.Format("02-01-2006 15:04") + "]" + "(" + feed.Items[count].Link + ")"
							tgBot.Send(tgCbMsg)
							break
						}
						tgCbMsg.Text = "[" + feed.Items[count].Title + "\n" + feed.Items[count].Date.Format("02-01-2006 15:04") + "]" + "(" + feed.Items[count].Link + ")"
						tgBot.Send(tgCbMsg)
					}
					continue
				}
				// Update visit time
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
				btF := "Главное меню"
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
				tgMsg.ReplyMarkup = keyboard
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
			case "beltsy":
				var city News
				city, err := NewsQuery(config.QueryTop)
				if err != nil {
					log.Println(err)
				}
				for _, cityItem := range city.Nodes {
					// log.Println(topItem.Node.NodeTitle, topItem.Node.NodePath)
					tgMsg.Text = "[" + cityItem.Node.NodeTitle + "]" + "(" + cityItem.Node.NodePath + ")"
					tgBot.Send(tgMsg)
				}
				tgMsg.Text = "_Оформив подиску на городские оповещения, Вы будете получать сюда предупреждения городских служб, анонсы мероприятий в Бельцах и т.д._"
				buttonSubscribe := tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribec")
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Нет, спасибо", "help")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSubscribe, buttonHelp))
				tgMsg.ReplyMarkup = keyboard
			case "top":
				var top News
				top, err := NewsQuery(config.QueryTop)
				if err != nil {
					log.Println(err)
				}
				for _, topItem := range top.Nodes {
					tgMsg.Text = "[" + topItem.Node.NodeTitle + "]" + "(" + topItem.Node.NodePath + ")"
					tgBot.Send(tgMsg)
				}
				tgMsg.Text = "_Хотите подписаться на самое популярное в \"СП\"? Мы будем присылать Вам такие подборки каждое воскресенье в 18:00_"
				buttonSubscribe := tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribetp")
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Нет, спасибо", "help")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSubscribe, buttonHelp))
				tgMsg.ReplyMarkup = keyboard
			case "news":
				buttonNext5 := tgbotapi.NewInlineKeyboardButtonData("Следующие "+strconv.Itoa(countView)+" новостей", "next5")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNext5))
				feed.Update()
				countFeed = 0
				for count, newsItem := range feed.Items {
					if count == countView-1 {
						countFeed = count
						tgMsg.ReplyMarkup = keyboard
						tgMsg.Text = "[" + newsItem.Title + "\n" + newsItem.Date.Format("02-01-2006 15:04") + "]" + "(" + newsItem.Link + ")"
						tgBot.Send(tgMsg)
						break
					}
					tgMsg.Text = "[" + newsItem.Title + "\n" + newsItem.Date.Format("02-01-2006 15:04") + "]" + "(" + newsItem.Link + ")"
					tgBot.Send(tgMsg)
				}
				continue
			case "search":
				tgMsg.Text = stubMsgText
			case "feedback":
				tgMsg.Text = stubMsgText
			case "holidays":
				if noWork {
					tgMsg.Text = stubMsgText
				} else {
					tgMsg.Text = "Молдавские, международные и религиозные праздники из нашего календаря	\"Существенный повод\" на ближайшую неделю:\n\n"
					for _, hd := range holidays {
						if (hd.Date.Unix() >= time.Now().AddDate(0, 0, -1).Unix()) && (hd.Date.Unix() <= time.Now().AddDate(0, 0, 7).Unix()) {
							tgMsg.Text += "*" + hd.Day + " " + hd.Month + "*" + "\n" + hd.Holiday + "\n\n"
						}
					}
					tgMsg.Text += "_Предлагаем Вам подписаться на рассылку праздников. Мы будем присылать Вам даты на неделю каждый понедельник в 10:00_"
					buttonSubscribe := tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribehd")
					buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Нет, спасибо", "help")
					keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSubscribe, buttonHelp))
					tgMsg.ReplyMarkup = keyboard
				}
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
