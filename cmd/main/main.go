package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/gorilla/websocket"
	"github.com/pgmod/envconfig"
)

var tgbot *bot.Bot
var TG_CHAT_IDs []int64

func main() {
	envconfig.Load()

	TG_BOT_TOKEN := envconfig.Get("TG_BOT_TOKEN", "")
	TG_CHAT_IDs = envconfig.GetInt64Slice("TG_CHAT_ID", []int64{})
	NTFY_ADDRS, err := envconfig.ToList(envconfig.Get("NTFY_ADDRS", ""), ",")
	if err != nil {
		log.Fatal(err)
	}

	if TG_BOT_TOKEN == "" {
		log.Fatal("TG_BOT_TOKEN is not set")
	}
	if len(TG_CHAT_IDs) == 0 {
		log.Fatal("TG_CHAT_ID is not set")
	}
	tgbot, _ = bot.New(TG_BOT_TOKEN)
	go tgbot.Start(context.Background())

	for _, chatID := range TG_CHAT_IDs {
		tgbot.SendMessage(
			context.Background(),
			&bot.SendMessageParams{
				ChatID: chatID,
				Text:   "Бот запущен для тем: \n" + strings.Join(NTFY_ADDRS, "\n"),
			},
		)
	}

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
	ID          string   `json:"id"`
	Time        int64    `json:"time"`
	Expires     int64    `json:"expires"`
	Event       string   `json:"event"`
	Topic       string   `json:"topic"`
	Message     string   `json:"message"`
	Priority    int      `json:"priority"`
	Tags        []string `json:"tags"`
	Title       string   `json:"title"`
	ContentType string   `json:"content_type"`
}

func listen(addr string) {
	fmt.Println("Listening to", addr)
	for {
		ws, err := dialNTFY(addr)
		if err != nil {
			log.Printf("[%s] connect failed: %v, retry in 5s", addr, err)
			time.Sleep(5 * time.Second)
			continue
		}

		for {
			messageType, data, err := ws.ReadMessage()
			if err != nil {
				log.Printf("[%s] read error: %v, reconnecting", addr, err)
				ws.Close()
				break
			}
			if messageType == websocket.CloseMessage {
				log.Printf("[%s] connection closed, reconnecting", addr)
				ws.Close()
				break
			}
			if messageType != websocket.TextMessage {
				continue
			}

			var msg Message
			fmt.Println(string(data))
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("[%s] unmarshal error: %v", addr, err)
				continue
			}
			if msg.Event != "message" {
				continue
			}

			var pm models.ParseMode
			if msg.ContentType == "text/markdown" {
				pm = models.ParseModeMarkdown
			}
			for _, chatID := range TG_CHAT_IDs {
				if _, err := tgbot.SendMessage(
					context.Background(),
					&bot.SendMessageParams{
						ChatID:    chatID,
						Text:      tagsToEmoji(msg.Tags) + " " + msg.Title + "\n" + escapeMsg(msg.Message),
						ParseMode: pm,
					},
				); err != nil {
					log.Printf("[%s] telegram send error (chat %d): %v", addr, chatID, err)
				}
			}
		}
	}
}

func dialNTFY(addr string) (*websocket.Conn, error) {
	ws, _, err := websocket.DefaultDialer.Dial("wss://"+addr+"/ws", nil)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func tagsToEmoji(tags []string) string {
	result := ""
	for _, tag := range tags {
		if emoji, ok := Emoji[tag]; ok {
			result += emoji
		}
	}
	fmt.Println("Emoji:", result)
	return result
}

func escapeMsg(msg string) string {
	msg = strings.ReplaceAll(msg, "_", "\\_")
	// msg = strings.ReplaceAll(msg, "*", "\\*")
	msg = strings.ReplaceAll(msg, "+", "\\+")
	msg = strings.ReplaceAll(msg, "[", "\\[")
	msg = strings.ReplaceAll(msg, "]", "\\]")
	msg = strings.ReplaceAll(msg, "(", "\\(")
	msg = strings.ReplaceAll(msg, ")", "\\)")
	msg = strings.ReplaceAll(msg, ".", "\\.")
	msg = strings.ReplaceAll(msg, "=", "\\=")
	msg = strings.ReplaceAll(msg, "-", "\\-")
	return msg
}
