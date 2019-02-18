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
	"strconv"

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
	// Start webhook
	vb := viber.New(botConfig.Bots.Viber.VBApikey, "IumasLink", "")
	vAccount, err := vb.AccountInfo()
	if err != nil {
		log.Println(err)
	}
	vb.Message = msgReceived
	vb.Sender.Avatar = vAccount.Icon
	http.Handle("/", vb)
	log.Println("Hello, I am ", vAccount.Name)
	go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(botConfig.Bots.Viber.VBPort), nil)
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
