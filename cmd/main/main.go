package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/telegram"
	"github.com/pgmod/envconfig"
)

func main() {
	envconfig.Load()

	TG_BOT_TOKEN := envconfig.Get("TG_BOT_TOKEN", "")
	TG_CHAT_ID := envconfig.GetInt64("TG_CHAT_ID", 0)
	NTFY_ADDRS, err := envconfig.ToList(envconfig.Get("NTFY_ADDRS", ""), ",")
	if err != nil {
		log.Fatal(err)
	}

	if TG_BOT_TOKEN == "" {
		log.Fatal("TG_BOT_TOKEN is not set")
	}
	if TG_CHAT_ID == 0 {
		log.Fatal("TG_CHAT_ID is not set")
	}
	telegramService, _ := telegram.New(TG_BOT_TOKEN)
	telegramService.AddReceivers(TG_CHAT_ID)

	notify.UseServices(telegramService)

	_ = notify.Send(
		context.Background(),
		"Запуск бота",
		"Бот запущен для тем: \n"+strings.Join(NTFY_ADDRS, "\n"),
	)

	wg := sync.WaitGroup{}
	for _, addr := range NTFY_ADDRS {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			listen(addr)
		}(addr)
	}
	wg.Wait()
}

type Message struct {
	ID      string `json:"id"`
	Time    int64  `json:"time"`
	Expires int64  `json:"expires"`
	Event   string `json:"event"`
	Topic   string `json:"topic"`
	Message string `json:"message"`
}

func listen(addr string) {
	fmt.Println("Listening to", addr)
	ws, _, _ := websocket.DefaultDialer.Dial("wss://"+addr+"/ws", nil)
	for {

		messageType, data, err := ws.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}
		var msg Message
		err = json.Unmarshal(data, &msg)
		if err != nil {
			log.Fatal(err)
		}
		if messageType == websocket.TextMessage {
			if msg.Event == "message" {
				err = notify.Send(
					context.Background(),
					msg.Topic,
					msg.Message,
				)
				if err != nil {
					log.Fatal(err)
				}
			}
		} else if messageType == websocket.CloseMessage {
			log.Println("Connection closed")
			ws, _, _ = websocket.DefaultDialer.Dial("wss://"+addr+"/ws", nil)
		}
	}
}
