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

	"github.com/mileusna/viber"
)

func main() {
	log.Println("Begin VB")
	v := viber.New("48799744e527d2c8-d447e6da2f634a31-8a717c2c7cd69d51", "IumasLink", "")
	vAccount, err := v.AccountInfo()
	if err != nil {
		log.Println(err)
	}
	v.Sender.Avatar = vAccount.Icon
	http.Handle("/", v)
	go http.ListenAndServe("0.0.0.0:8444", nil)
	webHookResp, err := v.SetWebhook("https://dev.infinitloop.md:8444/", nil)
	if err != nil {
		log.Println("WebHook error=> ", err)
	} else {
		log.Println("WebHook resp=> ", webHookResp)
	}

	userID := "bnzFlKadhfEx/nOKdHXrCw==" // fake user ID, use the real one
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
