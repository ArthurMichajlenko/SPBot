package main

import (
	"github.com/asdine/storm"
	"log"
	"time"
)

//VbUser ...
type VbUser struct {
	ID                string
	Username          string
	LastDate          time.Time
	Subscribe9        bool
	Subscribe20       bool
	SubscribeLast     bool
	SubscribeCity     bool
	SubscribeTop      bool
	SubscribeHolidays bool
}

//TgUser ...
type TgUser struct {
	ChatID            int64
	FirstName         string
	LastName          string
	Username          string
	LastDate          int
	Subscribe9        bool
	Subscribe20       bool
	SubscribeLast     bool
	SubscribeCity     bool
	SubscribeTop      bool
	SubscribeHolidays bool
	RssLastID         int
}

func main() {
	var tgusers []TgUser
	var vbusers []VbUser
	log.Println("Start...")
	dbTg, err := storm.Open("tguser.db")
	if err != nil {
		log.Println(err)
	}
	dbVb, err := storm.Open("vbuser.db")
	if err != nil {
		log.Println(err)
	}
	defer dbTg.Close()
	defer dbVb.Close()
	err = dbTg.All(&tgusers)
	if err != nil {
		log.Println(err)
	}
	err = dbVb.All(&vbusers)
	if err != nil {
		log.Println(err)
	}
	log.Println(tgusers)
	log.Println(vbusers)

}
