/*
 * Copyright (c) 2018 Arthur Michajlenko
 *
 * @Script: SPBot.go
 * @Author: Arthur Michajlenko
 * @Email: michajlenko1967@gmail.com
 * @Create At: 2018-04-04 15:25:00
 * @Last Modified By: Arthur Michajlenko
 * @Last Modified At: 2018-05-16 14:49:42
 * @Description: Bot for SP.
 */

package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Syfaro/telegram-bot-api"
	"github.com/asdine/storm"
	"github.com/fsnotify/fsnotify"
	"github.com/robfig/cron"
)

var (
	// botConfig configurations bot
	botConfig  Config
	mailAttach AttachFile
)

func main() {
	// Load botConfig
	var err error
	botConfig, err = LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	var (
		// Load holidays if error send message not released
		noWork = false
		// Message consist of few parts e.g. feedback, search
		numPageSearch     int
		numPageNews       int
		multipartFeedback = false
		multipartSearch   = false
		attachmentURLs    []string
		msgString         string
		searchString      string
		messageOwner      TgMessageOwner
		messageDate       time.Time
	)
	holidays, err := LoadHolidays(botConfig.FileHolidays)
	if err != nil {
		log.Println(err)
		noWork = true
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
	}
	defer watcher.Close()
	err = watcher.Add(botConfig.FileHolidays)
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
	tgBot, err := tgbotapi.NewBotAPI(botConfig.Bots.Telegram.TgApikey)
	if err != nil {
		log.Panic(err)
	}
	// TODO: Next 2 strings for development must remove in production
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
	_Вы можете прикрепите не более 5 файлов размером не более 20 MB каждый_
	*ВНИМАНИЕ* Все вложения должны отправляться как файл.
	/holidays - календарь праздников.
	/games - поиграть в игру.
	/donate - поддержать "СП".`
	startMsgEndText := `Спасибо за Ваш выбор! Вы можете отписаться от нашей рассылки в любой момент в меню /subscriptions`
	var ptgUpdates = new(tgbotapi.UpdatesChannel)
	tgUpdates := *ptgUpdates
	if botConfig.Bots.Telegram.TgWebhook == "" {
		// Initialize polling
		tgBot.RemoveWebhook()
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60
		tgUpdates, _ = tgBot.GetUpdatesChan(u)
	} else {
		// Initialize webhook & channel for update from API
		tgConURI := botConfig.Bots.Telegram.TgWebhook + ":" + strconv.Itoa(botConfig.Bots.Telegram.TgPort) + "/"
		tgBot.RemoveWebhook()
		_, err = tgBot.SetWebhook(tgbotapi.NewWebhook(tgConURI + tgBot.Token))
		if err != nil {
			log.Fatal(err)
		}
		// Listen Webhook
		tgUpdates = tgBot.ListenForWebhook("/" + tgBot.Token)
		go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(botConfig.Bots.Telegram.TgPort), nil)
	}
	// Cron for subscriptions
	c := cron.New()
	c.AddFunc("0 0/15 * * * *", func() {
		// tg40Msg := tgbotapi.NewMessage(474165300, startMsgText)
		// tg40Msg.ParseMode = "Markdown"
		// tgBot.Send(tg40Msg)
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
			holidays, err = LoadHolidays(botConfig.FileHolidays)
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
					multipartFeedback = false
					multipartSearch = false
					attachmentURLs = nil
					mailAttach.FileName = nil
					mailAttach.ContentType = nil
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
				case "search":
					var search News
					search, err := SearchQuery(searchString, numPageSearch)
					if err != nil {
						log.Println(err)
					}
					if len(search.Nodes) == 0 {
						tgCbMsg.Text = "По Вашему запросу ничего не найдено"
						// tgBot.Send(tgCbMsg)
						multipartSearch = false
						break
					} else {
						for _, searchItem := range search.Nodes {
							tgCbMsg.Text = searchItem.Node.NodeDate + "\n[" + searchItem.Node.NodeTitle + "]" + "(" + searchItem.Node.NodePath + ")"
							tgBot.Send(tgCbMsg)
						}
					}
					multipartSearch = false
					buttonSearchNext := tgbotapi.NewInlineKeyboardButtonData("Вперед", "searchnext")
					keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSearchNext))
					tgCbMsg.ReplyMarkup = keyboard
					tgCbMsg.Text = "Страница: " + strconv.Itoa(numPageSearch+1)
				case "searchnext":
					numPageSearch++
					var search News
					search, err := SearchQuery(searchString, numPageSearch)
					if err != nil {
						log.Println(err)
					}
					for _, searchItem := range search.Nodes {
						tgCbMsg.Text = searchItem.Node.NodeDate + "\n[" + searchItem.Node.NodeTitle + "]" + "(" + searchItem.Node.NodePath + ")"
						tgBot.Send(tgCbMsg)
					}
					multipartSearch = false
					buttonSearchNext := tgbotapi.NewInlineKeyboardButtonData("Вперед", "searchnext")
					buttonSearchPrev := tgbotapi.NewInlineKeyboardButtonData("Назад", "searchprev")
					if len(search.Nodes) != 0 {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSearchPrev, buttonSearchNext))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Страница: " + strconv.Itoa(numPageSearch+1)
					} else {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSearchPrev))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Вы в конце поиска"
					}
				case "searchprev":
					numPageSearch--
					var search News
					search, err := SearchQuery(searchString, numPageSearch)
					if err != nil {
						log.Println(err)
					}
					for _, searchItem := range search.Nodes {
						tgCbMsg.Text = searchItem.Node.NodeDate + "\n[" + searchItem.Node.NodeTitle + "]" + "(" + searchItem.Node.NodePath + ")"
						tgBot.Send(tgCbMsg)
					}
					multipartSearch = false
					buttonSearchNext := tgbotapi.NewInlineKeyboardButtonData("Вперед", "searchnext")
					buttonSearchPrev := tgbotapi.NewInlineKeyboardButtonData("Назад", "searchprev")
					if numPageSearch != 0 {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSearchPrev, buttonSearchNext))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Страница: " + strconv.Itoa(numPageSearch+1)
					} else {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSearchNext))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Вы в начале поиска"
					}
				case "newsnext":
					numPageNews++
					urlNews := botConfig.QueryNews24H
					news, err := NewsQuery(urlNews, numPageNews)
					if err != nil {
						log.Println(err)
					}
					for _, newsItem := range news.Nodes {
						tgCbMsg.Text = newsItem.Node.NodeDate + "\n[" + newsItem.Node.NodeTitle + "]" + "(" + newsItem.Node.NodePath + ")"
						tgBot.Send(tgCbMsg)
					}
					buttonNewsPrev := tgbotapi.NewInlineKeyboardButtonData("Предидущие 10 новостей", "newsprev")
					buttonNewsNext := tgbotapi.NewInlineKeyboardButtonData("Следующие 10 новостей", "newsnext")
					if len(news.Nodes) != 0 {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNewsPrev, buttonNewsNext))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Страница: " + strconv.Itoa(numPageNews+1)
					} else {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNewsPrev))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Больше новостей нет"
					}
				case "newsprev":
					numPageNews--
					urlNews := botConfig.QueryNews24H
					news, err := NewsQuery(urlNews, numPageNews)
					if err != nil {
						log.Println(err)
					}
					for _, newsItem := range news.Nodes {
						tgCbMsg.Text = newsItem.Node.NodeDate + "\n[" + newsItem.Node.NodeTitle + "]" + "(" + newsItem.Node.NodePath + ")"
						tgBot.Send(tgCbMsg)
					}
					buttonNewsPrev := tgbotapi.NewInlineKeyboardButtonData("Предидущие 10 новостей", "newsprev")
					buttonNewsNext := tgbotapi.NewInlineKeyboardButtonData("Следующие 10 новостей", "newsnext")
					if numPageNews != 0 {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNewsPrev, buttonNewsNext))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Страница: " + strconv.Itoa(numPageNews+1)
					} else {
						keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNewsNext))
						tgCbMsg.ReplyMarkup = keyboard
						tgCbMsg.Text = "Последние новости"
					}
				case "sendfeedback":
					emailSubject := "Telegram\n"
					emailSubject += "Сообщение от: ID:" + messageOwner.ID + " Username: " + messageOwner.Username + "\n"
					emailSubject += "Имя Фамилия: " + messageOwner.FirstName + " " + messageOwner.LastName + "\n"
					emailSubject += "Дата: " + messageDate.String()
					go func(emailSubject string, msgString string, attachmentURLs []string, fileName []string, contentType []string) {
						err := SendFeedback(emailSubject, msgString, attachmentURLs, fileName, contentType)
						if err != nil {
							log.Printf("Send Feedback err: %#+v\n", err)
						}
					}(emailSubject, msgString, attachmentURLs, mailAttach.FileName, mailAttach.ContentType)
					attachmentURLs = nil
					mailAttach.FileName = nil
					mailAttach.ContentType = nil
					multipartFeedback = false
					tgCbMsg.Text = `Ваше сообщение отправлено. Спасибо `
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
			if !tgUpdate.Message.IsCommand() && !multipartFeedback && !multipartSearch {
				tgMsg.ReplyToMessageID = tgUpdate.Message.MessageID
				tgMsg.Text = noCmdText
				tgMsg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
				tgBot.Send(tgMsg)
				continue
			}

			msgSlice := strings.Split(tgUpdate.Message.Text, " ")
			switch strings.ToLower(msgSlice[0]) {
			// switch tgUpdate.Message.Command() {
			case "/help":
				tgMsg.Text = helpMsgText
			case "/start":
				tgMsg.Text = startMsgText
				buttonSubscribe := tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribestart")
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Нет, спасибо", "help")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSubscribe, buttonHelp))
				tgMsg.ReplyMarkup = keyboard
			case "/subscriptions":
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
			case "/beltsy":
				var city News
				numPage := 1
				queryCity := botConfig.QueryTop + "page=" + strconv.Itoa(numPage)
				city, err := NewsQuery(queryCity, 0)
				if err != nil {
					log.Println(err)
				}
				for _, cityItem := range city.Nodes {
					tgMsg.Text = "[" + cityItem.Node.NodeTitle + "]" + "(" + cityItem.Node.NodePath + ")"
					tgBot.Send(tgMsg)
				}
				tgMsg.Text = "_Оформив подиску на городские оповещения, Вы будете получать сюда предупреждения городских служб, анонсы мероприятий в Бельцах и т.д._"
				buttonSubscribe := tgbotapi.NewInlineKeyboardButtonData("Подписаться", "subscribec")
				buttonHelp := tgbotapi.NewInlineKeyboardButtonData("Нет, спасибо", "help")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSubscribe, buttonHelp))
				tgMsg.ReplyMarkup = keyboard
			case "/top":
				var top News
				numPage := 1
				queryTop := botConfig.QueryTop + "page=" + strconv.Itoa(numPage)
				top, err := NewsQuery(queryTop, 0)
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
			case "/news":
				numPageNews = 0
				urlNews := botConfig.QueryNews24H
				news, err := NewsQuery(urlNews, numPageNews)
				if err != nil {
					log.Println(err)
				}
				for _, newsItem := range news.Nodes {
					tgMsg.Text = newsItem.Node.NodeDate + "\n[" + newsItem.Node.NodeTitle + "]" + "(" + newsItem.Node.NodePath + ")"
					tgBot.Send(tgMsg)
				}
				buttonNewsNext := tgbotapi.NewInlineKeyboardButtonData("Следующие 10 новостей", "newsnext")
				keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonNewsNext))
				tgMsg.ReplyMarkup = keyboard
			case "/search":
				multipartSearch = true
				numPageSearch = 0
				tgMsg.Text = "Введите что искать"
			case "/feedback":
				multipartFeedback = true
				messageOwner.ID = strconv.Itoa(int(tgUpdate.Message.Chat.ID))
				messageOwner.Username = tgUpdate.Message.Chat.UserName
				messageOwner.FirstName = tgUpdate.Message.Chat.FirstName
				messageOwner.LastName = tgUpdate.Message.Chat.LastName
				messageDate = tgUpdate.Message.Time()
				tgMsg.Text = "Введите текст сообщения..."
			case "/holidays":
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
			case "/games":
				tgMsg.Text = stubMsgText
			case "/donate":
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
				switch {
				case multipartFeedback:
					if tgUpdate.Message.Document != nil {
						if len(attachmentURLs) != 5 {
							mailAttach.BotFile = tgbotapi.File{FileID: tgUpdate.Message.Document.FileID, FileSize: tgUpdate.Message.Document.FileSize}
							mailAttach.FileName = append(mailAttach.FileName, tgUpdate.Message.Document.FileName)
							mailAttach.ContentType = append(mailAttach.ContentType, tgUpdate.Message.Document.MimeType)
							urlAttach, _ := tgBot.GetFileDirectURL(mailAttach.BotFile.FileID)
							attachmentURLs = append(attachmentURLs, urlAttach)
							tgMsg.Text = "Вы можете приложить еще *" + strconv.Itoa(5-len(attachmentURLs)) + "* файл(а)\n*ВНИМАНИЕ* _Все вложения должны отправляться как файлы. Размер файла не должен превышать 20 MB_"
							tgBot.Send(tgMsg)
						} else {
							tgMsg.Text = "_Вы исчерпали количество вложений_"
							tgBot.Send(tgMsg)
						}
					} else {
						tgMsg.Text = `Вы можете приложить до 5 файлов к своему сообщению.
						*ВНИМАНИЕ* _Все вложения должны отправляться как файлы. Размер файла не должен превышать 20 MB_`
						tgBot.Send(tgMsg)
						msgString = tgUpdate.Message.Text
					}
					buttonYes := tgbotapi.NewInlineKeyboardButtonData("Да", "sendfeedback")
					buttonNo := tgbotapi.NewInlineKeyboardButtonData("Нет", "help")
					keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonYes, buttonNo))
					tgMsg.ReplyMarkup = keyboard
					tgMsg.Text = "Отправить сообщение?"
				case multipartSearch:
					searchString = tgUpdate.Message.Text
					buttonSearch := tgbotapi.NewInlineKeyboardButtonData("Искать", "search")
					buttonEscape := tgbotapi.NewInlineKeyboardButtonData("Отменить", "help")
					keyboard := tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttonSearch, buttonEscape))
					tgMsg.ReplyMarkup = keyboard
					tgMsg.Text = "Начинаем поиск ..."
					// tgBot.Send(tgMsg)
				default:
					toOriginal = true
					tgMsg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
					tgMsg.Text = noCmdText
				}
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
			// ! Default may cause high CPU load
			// default:
		}
	}
}
