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

	"github.com/mileusna/viber"
)

var botConfig Config

func main() {
	// Load botConfig
	var err error
	botConfig, err = LoadConfigBots("config.json")
	if err != nil {
		log.Panic(err)
	}
	// Start webhook
	v := viber.New(botConfig.Bots.Viber.VBApikey, "IumasLink", "")
	vAccount, err := v.AccountInfo()
	if err != nil {
		log.Println(err)
	}
	v.Sender.Avatar = vAccount.Icon
	http.Handle("/", v)
	log.Println("Hello, I am ", vAccount.Name)
	go http.ListenAndServe("0.0.0.0:"+strconv.Itoa(botConfig.Bots.Viber.VBPort), nil)
	webHookResp, err := v.SetWebhook(botConfig.Bots.Viber.VBWebhook+":"+strconv.Itoa(botConfig.Bots.Viber.VBPort), nil)
	if err != nil {
		log.Println("WebHook error=> ", err)
	} else {
		log.Println("WebHook resp=> ", webHookResp)
	}

	userID := "bnzFlKadhfEx/nOKdHXrCw==" // My User ID
	// send text message
	token, err := v.SendTextMessage(userID, "Hello, World!\nПривет Мир")
	if err != nil {
		log.Println("Viber error:", err)
	} else {
		log.Println("Message sent, message token:", token)
	}
	log.Println(v)
	log.Println(vAccount)
}
