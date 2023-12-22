package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var bot *tgbotapi.BotAPI
var client *genai.Client

func getFileBytes(src string) ([]byte, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxConnsPerHost:     50,
			MaxIdleConnsPerHost: 50,
			MaxIdleConns:        100,
		},
	}

	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func main() {
	ctx := context.Background()
	var err error
	bot, err = tgbotapi.NewBotAPI(os.Getenv("Telegram_Bot_Key"))
	if err != nil {
		log.Panic(err)
	}
	bot.Debug = false
	log.Printf("Authorized on account %s", bot.Self.UserName)

	client, err = genai.NewClient(ctx, option.WithAPIKey(os.Getenv("API_KEY")))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	waitForBot(ctx)
	select {}
}
func waitForBot(ctx context.Context) {

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	var msg tgbotapi.MessageConfig
	for update := range updates {
		if update.Message != nil { // If we got a message

			if len(update.Message.Photo) > 0 {

				file, err := bot.GetFileDirectURL(update.Message.Photo[len(update.Message.Photo)-1].FileID)
				imgData, err := getFileBytes(file)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %s", err))
					bot.Send(msg)
					continue
				}
				prompt := []genai.Part{
					genai.ImageData("jpeg", imgData),
					genai.Text(update.Message.Text),
				}
				model := client.GenerativeModel("gemini-pro-vision")
				resp, err := model.GenerateContent(ctx, prompt...)
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %s", err))
					bot.Send(msg)
					continue
				}
				q := resp.Candidates[0].Content.Parts[0]
				markdown := fmt.Sprintf("%s", q)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, markdown)
				msg.ParseMode = tgbotapi.ModeMarkdown
				msg.Entities = update.Message.Entities
				bot.Send(msg)
			} else {

				model := client.GenerativeModel("gemini-pro")
				resp, err := model.GenerateContent(ctx, genai.Text(update.Message.Text))
				if err != nil {
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Error: %s", err))
					bot.Send(msg)
					continue
				}
				q := resp.Candidates[0].Content.Parts[0]
				markdown := fmt.Sprintf("%s", q)
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, markdown)
				msg.ParseMode = tgbotapi.ModeMarkdown
				msg.Entities = update.Message.Entities
				bot.Send(msg)
			}

		}
	}
}
