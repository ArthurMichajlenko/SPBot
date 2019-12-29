/*
 * Copyright (c) 2018 Arthur Michajlenko
 *
 * @Script: VBBot.go
 * @Author: Arthur Michajlenko
 * @Email: michajlenko1967@gmail.com
 * @Create At: 2018-09-17 10:49:45
 * @Last Modified By: Arthur Michajlenko
 * @Last Modified At: 2018-09-17 11:26:39
 * @Description: Viber bot.
 */

package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/asdine/storm"
	"github.com/robfig/cron"

	"github.com/fsnotify/fsnotify"

	"github.com/mileusna/viber"
)

var (
	botConfig Config
	//NoWork is Holidays work or not
	NoWork = false
	//HolidayList slice of holidays
	HolidayList []Holidays
)

func main() {
	// Load botConfig
	var err error
	botConfig, err = LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	//Log
	if !botConfig.Debug {
		logfile, err := os.OpenFile(botConfig.Bots.Viber.LogFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Println(err)
		}
		defer logfile.Close()
		log.SetOutput(logfile)
	}
	// Start webhook
	vb := viber.New(botConfig.Bots.Viber.VBApikey, "Новостной бот СП", "")
	vAccount, err := vb.AccountInfo()
	if err != nil {
		log.Println(err)
	}
	vb.Message = msgReceived
	vb.ConversationStarted = msgConversationStarted
	vb.Subscribed = msgSubscribed
	vb.Sender.Avatar = vAccount.Icon
	http.Handle("/", vb)
	log.Println("Hello, I am ", vAccount.Name)
	// For development ListenAndServe, for work ListenAndServeTLS
	// go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(botConfig.Bots.Viber.VBPort), nil)
	go http.ListenAndServeTLS("0.0.0.0:"+strconv.Itoa(botConfig.Bots.Viber.VBPort), botConfig.Bots.Viber.VBPathCERT, botConfig.Bots.Viber.VBPathKey, nil)
	webHookResp, err := vb.SetWebhook(botConfig.Bots.Viber.VBWebhook+":"+strconv.Itoa(botConfig.Bots.Viber.VBPort), nil)
	if err != nil {
		log.Println("WebHook error=> ", err)
	} else {
		log.Println("WebHook resp=> ", webHookResp)
	}

	// send text message
	// userID := "bnzFlKadhfEx/nOKdHXrCw==" // My User ID
	// token, err := vb.SendTextMessage(userID, "Hello, World!\nПривет Мир")
	// if err != nil {
	// 	log.Println("Viber error:", err)
	// } else {
	// 	log.Println("Message sent, message token:", token)
	// }
	log.Println(vb)
	log.Println(vAccount)

	//Holidays file handler
	HolidayList, err = LoadHolidays(botConfig.FileHolidays)
	if err != nil {
		log.Println(err)
		NoWork = true
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
	// Cron for subscriptions
	c := cron.New()
	//Top subscribe
	c.AddFunc("0 55 17 * * 0", func() {
		var vbbusers []VbUser
		var topc News
		var topv News
		urlTopC := botConfig.QueryTopComments
		urlTopV := botConfig.QueryTopViews
		topc, err := NewsQuery(urlTopC, -1)
		if err != nil {
			log.Println(err)
		}
		topv, err = NewsQuery(urlTopV, -1)
		if err != nil {
			log.Println(err)
		}
		db, err := storm.Open("vbuser.db")
		if err != nil {
			log.Println(err)
		}
		defer db.Close()
		db.Find("SubscribeTop", true, &vbbusers)
		for _, subUser := range vbbusers {
			msgCarouselComment := vb.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselView := vb.NewRichMediaMessage(6, 7, spColorBG)
			vb.SendTextMessage(subUser.ID, "Самые читаемые за последние семь дней")
			for _, topItem := range topv.Nodes {
				msgCarouselView.AddButton(vb.NewTextButton(6, 2, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeDate+"\n"+topItem.Node.NodeTitle))
				msgCarouselView.AddButton(vb.NewImageButton(6, 4, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeCover["src"]))
				msgCarouselView.AddButton(vb.NewTextButton(6, 1, viber.OpenURL, topItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			vb.SendMessage(subUser.ID, msgCarouselView)
			vb.SendTextMessage(subUser.ID, "Самые комментируемые за последние семь дней")
			for _, topItem := range topc.Nodes {
				msgCarouselComment.AddButton(vb.NewTextButton(6, 2, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeDate+"\n"+topItem.Node.NodeTitle))
				msgCarouselComment.AddButton(vb.NewImageButton(6, 4, viber.OpenURL, topItem.Node.NodePath, topItem.Node.NodeCover["src"]))
				msgCarouselComment.AddButton(vb.NewTextButton(6, 1, viber.OpenURL, topItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			vb.SendMessage(subUser.ID, msgCarouselComment)
			msgNavig := vb.NewRichMediaMessage(6, 2, "#ffffff")
			msgNavig.AddButton(vb.NewTextButton(6, 2, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			vb.SendMessage(subUser.ID, msgNavig)
		}
	})
	//Holidays subscribe
	c.AddFunc("0 02 10 * * 1", func() {
		var vbbusers []VbUser
		var msgText string
		if NoWork {
			return
		}
		db, err := storm.Open("vbuser.db")
		if err != nil {
			log.Println(err)
		}
		defer db.Close()
		db.Find("SubscribeHolidays", true, &vbbusers)
		for _, subUser := range vbbusers {
			msgText = "Молдавские, международные и религиозные праздники из нашего календаря	\"Существенный Повод\" на ближайшую неделю:\n\n"
			for _, hd := range HolidayList {
				day, _ := strconv.Atoi(hd.Day)
				if (hd.Date.Unix() >= time.Now().AddDate(0, 0, -1).Unix()) && (hd.Date.Unix() <= time.Now().AddDate(0, 0, 7).Unix()) {
					msgText += "*" + strconv.Itoa(day) + " " + hd.Month + "*" + "\n" + hd.Holiday + "\n\n"
				}
			}
			vb.SendMessage(subUser.ID, vb.NewTextMessage(msgText))
			msgNavig := vb.NewRichMediaMessage(6, 2, "#ffffff")
			msgNavig.AddButton(vb.NewTextButton(6, 2, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			vb.SendMessage(subUser.ID, msgNavig)
		}
	})
	//News subscribe
	c.AddFunc("@hourly", func() {
		var vbbusers []VbUser
		var lastNews News
		urlLast := botConfig.QueryNews1H
		lastNews, err = NewsQuery(urlLast, 0)
		if err != nil {
			log.Println("Query: ",err)
		}
		if len(lastNews.Nodes) == 0 {
			return
		}
		db, err := storm.Open("vbuser.db")
		if err != nil {
			log.Println("Database: ",err)
		}
		defer db.Close()
		db.Find("SubscribeLast", true, &vbbusers)
		for _, subUser := range vbbusers {
			msgCarouselNews := vb.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselNews1 := vb.NewRichMediaMessage(6, 7, spColorBG)
			vb.SendTextMessage(subUser.ID, "Последние новости")
			for i, newsItem := range lastNews.Nodes {
				srcDate := newsItem.Node.NodeDate
				if i < 5 {
					msgCarouselNews.AddButton(vb.NewTextButton(6, 2, viber.OpenURL, newsItem.Node.NodePath, strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[1]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[0]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[2]+strings.SplitAfter(srcDate, " ")[3]+"\n"+newsItem.Node.NodeTitle))
					msgCarouselNews.AddButton(vb.NewImageButton(6, 4, viber.OpenURL, newsItem.Node.NodePath, newsItem.Node.NodeCover["src"]))
					msgCarouselNews.AddButton(vb.NewTextButton(6, 1, viber.OpenURL, newsItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
				} else {
					msgCarouselNews1.AddButton(vb.NewTextButton(6, 2, viber.OpenURL, newsItem.Node.NodePath, strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[1]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[0]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[2]+strings.SplitAfter(srcDate, " ")[3]+"\n"+newsItem.Node.NodeTitle))
					msgCarouselNews1.AddButton(vb.NewImageButton(6, 4, viber.OpenURL, newsItem.Node.NodePath, newsItem.Node.NodeCover["src"]))
					msgCarouselNews1.AddButton(vb.NewTextButton(6, 1, viber.OpenURL, newsItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
				}
			}
			vb.SendMessage(subUser.ID, msgCarouselNews)
			vb.SendMessage(subUser.ID, msgCarouselNews1)
			msgNavig := vb.NewRichMediaMessage(6, 2, "#ffffff")
			msgNavig.AddButton(vb.NewTextButton(6, 2, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			vb.SendMessage(subUser.ID, msgNavig)
		}
	})
	//09:00 subscribe
	c.AddFunc("0 03 09 * * *", func() {
		Page = 0
		var vbbusers []VbUser
		var lastNews News
		var rangeNews []NodeNews
	NewsBreak:
		for {
			urlLast := botConfig.QueryNews24H
			lastNews, err = NewsQuery(urlLast, Page)
			if err != nil {
				log.Println(err)
			}
			if len(lastNews.Nodes) == 0 {
				return
			}
			for _, itemRangeNews := range lastNews.Nodes {
				if CheckNewsRange(itemRangeNews.Node.NodeDate) {
					rangeNews = append(rangeNews, itemRangeNews.Node)
				} else {
					break NewsBreak
				}
			}
			Page++
		}
		db, err := storm.Open("vbuser.db")
		if err != nil {
			log.Println(err)
		}
		defer db.Close()
		db.Find("Subscribe9", true, &vbbusers)
		for _, subUser := range vbbusers {
			var rangeMsgCarousel []*viber.RichMediaMessage
			msgNavig := vb.NewRichMediaMessage(6, 2, "#ffffff")
			vb.SendTextMessage(subUser.ID, "Материалы за последние сутки")
			c := 0
			cc := 0
			rangeMsgCarousel = append(rangeMsgCarousel, vb.NewRichMediaMessage(6, 7, spColorBG))
			rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 2, viber.OpenURL, rangeNews[0].NodePath, rangeNews[0].NodeDate+"\n"+rangeNews[0].NodeTitle))
			rangeMsgCarousel[cc].AddButton(vb.NewImageButton(6, 4, viber.OpenURL, rangeNews[0].NodePath, rangeNews[0].NodeCover["src"]))
			rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 1, viber.OpenURL, rangeNews[0].NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			for m := 1; m < len(rangeNews); m++ {
				if c < 4 {
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 2, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeDate+"\n"+rangeNews[m].NodeTitle))
					rangeMsgCarousel[cc].AddButton(vb.NewImageButton(6, 4, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeCover["src"]))
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 1, viber.OpenURL, rangeNews[m].NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
					c++
				} else {
					c = 0
					cc++
					rangeMsgCarousel = append(rangeMsgCarousel, vb.NewRichMediaMessage(6, 7, spColorBG))
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 2, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeDate+"\n"+rangeNews[m].NodeTitle))
					rangeMsgCarousel[cc].AddButton(vb.NewImageButton(6, 4, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeCover["src"]))
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 1, viber.OpenURL, rangeNews[m].NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
				}
			}
			for _, news := range rangeMsgCarousel {
				vb.SendMessage(subUser.ID, news)
			}
			msgNavig.AddButton(vb.NewTextButton(6, 2, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			vb.SendMessage(subUser.ID, msgNavig)
		}
	})
	//20:00 subscribe
	c.AddFunc("0 03 20 * * *", func() {
		Page = 0
		var vbbusers []VbUser
		var lastNews News
		var rangeNews []NodeNews
	NewsBreak:
		for {
			urlLast := botConfig.QueryNews24H
			lastNews, err = NewsQuery(urlLast, Page)
			if err != nil {
				log.Println(err)
			}
			if len(lastNews.Nodes) == 0 {
				return
			}
			for _, itemRangeNews := range lastNews.Nodes {
				if CheckNewsRange(itemRangeNews.Node.NodeDate) {
					rangeNews = append(rangeNews, itemRangeNews.Node)
				} else {
					break NewsBreak
				}
			}
			Page++
		}
		db, err := storm.Open("vbuser.db")
		if err != nil {
			log.Println(err)
		}
		defer db.Close()
		db.Find("Subscribe20", true, &vbbusers)
		for _, subUser := range vbbusers {
			var rangeMsgCarousel []*viber.RichMediaMessage
			msgNavig := vb.NewRichMediaMessage(6, 2, "#ffffff")
			vb.SendTextMessage(subUser.ID, "Материалы за последние сутки")
			c := 0
			cc := 0
			rangeMsgCarousel = append(rangeMsgCarousel, vb.NewRichMediaMessage(6, 7, spColorBG))
			rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 2, viber.OpenURL, rangeNews[0].NodePath, rangeNews[0].NodeDate+"\n"+rangeNews[0].NodeTitle))
			rangeMsgCarousel[cc].AddButton(vb.NewImageButton(6, 4, viber.OpenURL, rangeNews[0].NodePath, rangeNews[0].NodeCover["src"]))
			rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 1, viber.OpenURL, rangeNews[0].NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			for m := 1; m < len(rangeNews); m++ {
				if c < 4 {
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 2, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeDate+"\n"+rangeNews[m].NodeTitle))
					rangeMsgCarousel[cc].AddButton(vb.NewImageButton(6, 4, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeCover["src"]))
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 1, viber.OpenURL, rangeNews[m].NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
					c++
				} else {
					c = 0
					cc++
					rangeMsgCarousel = append(rangeMsgCarousel, vb.NewRichMediaMessage(6, 7, spColorBG))
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 2, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeDate+"\n"+rangeNews[m].NodeTitle))
					rangeMsgCarousel[cc].AddButton(vb.NewImageButton(6, 4, viber.OpenURL, rangeNews[m].NodePath, rangeNews[m].NodeCover["src"]))
					rangeMsgCarousel[cc].AddButton(vb.NewTextButton(6, 1, viber.OpenURL, rangeNews[m].NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
				}
			}
			for _, news := range rangeMsgCarousel {
				vb.SendMessage(subUser.ID, news)
			}
			msgNavig.AddButton(vb.NewTextButton(6, 2, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			vb.SendMessage(subUser.ID, msgNavig)
		}
	})
	//City subscribe
	c.AddFunc("0 01 * * * *", func() {
		var vbbusers []VbUser
		var cityA News
		var cityD News
		urlCityA := botConfig.QueryCityAfisha
		cityA, err = NewsQuery(urlCityA, 0)
		if err != nil {
			log.Println(err)
		}
		urlCityD := botConfig.QueryCityDisp
		cityD, err = NewsQuery(urlCityD, 0)
		if err != nil {
			log.Println(err)
		}
		if (len(cityA.Nodes) == 0) && (len(cityD.Nodes) == 0) {
			return
		}
		db, err := storm.Open("vbuser.db")
		if err != nil {
			log.Println(err)
		}
		defer db.Close()
		db.Find("SubscribeCity", true, &vbbusers)
		for _, subUser := range vbbusers {
			msgCarouselCityA := vb.NewRichMediaMessage(6, 7, spColorBG)
			msgCarouselCityD := vb.NewRichMediaMessage(6, 7, spColorBG)
			vb.SendTextMessage(subUser.ID, "Городские оповещения")
			for _, newsItem := range cityA.Nodes {
				srcDate := newsItem.Node.NodeDate
				msgCarouselCityA.AddButton(vb.NewTextButton(6, 2, viber.OpenURL, newsItem.Node.NodePath, strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[1]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[0]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[2]+strings.SplitAfter(srcDate, " ")[3]+"\n"+newsItem.Node.NodeTitle))
				msgCarouselCityA.AddButton(vb.NewImageButton(6, 4, viber.OpenURL, newsItem.Node.NodePath, newsItem.Node.NodeCover["src"]))
				msgCarouselCityA.AddButton(vb.NewTextButton(6, 1, viber.OpenURL, newsItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			for _, newsItem := range cityD.Nodes {
				srcDate := newsItem.Node.NodeDate
				msgCarouselCityD.AddButton(vb.NewTextButton(6, 2, viber.OpenURL, newsItem.Node.NodePath, strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[1]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[0]+"."+strings.Split(strings.SplitAfter(srcDate, " ")[1], "/")[2]+strings.SplitAfter(srcDate, " ")[3]+"\n"+newsItem.Node.NodeTitle))
				msgCarouselCityD.AddButton(vb.NewImageButton(6, 4, viber.OpenURL, newsItem.Node.NodePath, newsItem.Node.NodeCover["src"]))
				msgCarouselCityD.AddButton(vb.NewTextButton(6, 1, viber.OpenURL, newsItem.Node.NodePath, `<font color="#ffffff">Подробнее...</font>`).SetBgColor(spColorBG))
			}
			vb.SendMessage(subUser.ID, msgCarouselCityA)
			vb.SendMessage(subUser.ID, msgCarouselCityD)
			msgNavig := vb.NewRichMediaMessage(6, 2, "#ffffff")
			msgNavig.AddButton(vb.NewTextButton(6, 2, viber.Reply, "menu", `<font color="#ffffff">Главное меню</font>`).SetBgColor(spColorBG).SetSilent())
			vb.SendMessage(subUser.ID, msgNavig)
		}
	})
	c.Start()
	//Get Updates from chanells
	for {
		select {
		case event := <-watcher.Events:
			log.Println("event: ", event)
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file: ", event.Name)
			}
			HolidayList, err = LoadHolidays(botConfig.FileHolidays)
			if err != nil {
				log.Println(err)
				NoWork = true
			} else {
				NoWork = false
			}
		case errEv := <-watcher.Errors:
			log.Println("error: ", errEv)
		}
	}
}
