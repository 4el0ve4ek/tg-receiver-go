package telegram

import (
	"log"
	"os"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"tg-receiver-bot/model"
	"tg-receiver-bot/receivers"
	"tg-receiver-bot/store"
)

type Bot struct {
	controller []receivers.Controller
	messages   chan *model.TelegramMessage
	done       chan struct{}
}

// New generate tg bot and all bot-receivers for him
func New() *Bot {
	messageChan := make(chan *model.TelegramMessage)
	VkController := receivers.NewVkBot(messageChan, store.NewTgVk())

	return &Bot{
		controller: []receivers.Controller{VkController},
		messages:   messageChan,
		done:       make(chan struct{}),
	}
}

// Start working process
func (b *Bot) Start() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("tg_token"))
	if err != nil {
		panic(err)
	}

	//bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	go b.observe(bot)

	go b.checkUpdates(bot)
}

// Stop working of bot
func (b *Bot) Stop() {
	close(b.done)
}

// waits for messages and send them to TG
func (b *Bot) observe(bot *tgbotapi.BotAPI) {
	for {
		select {
		case msg := <-b.messages:
			for _, tgMsg := range ConvertModel(msg) {
				_, err := bot.Send(tgMsg)
				if err != nil {
					log.Print(err)
				}
			}
		case <-b.done:
			log.Print("Stops sending message")
			return
		}
	}
}

// receives messages
func (b *Bot) checkUpdates(tg *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 45

	updates := tg.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		text := update.Message.Text
		command, link := separate(text)

		chatID := update.Message.Chat.ID

		var curController receivers.Controller

		for _, controller := range b.controller {
			if controller.Validate(link) {
				curController = controller
			}
		}

		if curController == nil {
			tg.Send(tgbotapi.NewMessage(chatID, "Invalid link or command. Please read instructions first."))
			continue
		}

		var response string
		switch command {
		case "/link":
			response = curController.Link(link, chatID)
		case "/unlink":
			response = curController.UnLink(link, chatID)
		default:
			response = "Invalid command. Read instructions first"
		}

		tg.Send(tgbotapi.NewMessage(chatID, response))
	}
}

// ConvertModel convert model.TelegramMessage to slice of messages which could be sent to TG
// some restrictions: for now it could be approx 5 text message and no more than 10 pictures,
//					  so it should work fast enough
// TODO: refactor this, cause its quite ugly
func ConvertModel(modelMessage *model.TelegramMessage) []tgbotapi.Chattable {
	modelMessage.Normalize()
	messages := make([]tgbotapi.Chattable, 0)
	for _, chatId := range modelMessage.ChatIds {
		for _, text := range modelMessage.Text {
			messages = append(messages, tgbotapi.NewMessage(chatId, text))
		}
		for _, photoUrl := range modelMessage.Photos {
			photoFile := tgbotapi.FileURL(photoUrl)
			messages = append(messages, tgbotapi.NewPhoto(chatId, photoFile))
		}
		for _, gifUrl := range modelMessage.Gif {
			gifFile := tgbotapi.FileURL(gifUrl)
			messages = append(messages, tgbotapi.NewAnimation(chatId, gifFile))
		}

	}
	return messages
}

// splits the message for command part and link part
func separate(text string) (command string, link string) {
	words := strings.SplitN(text, " ", 2)
	if len(words) < 2 {
		return words[0], ""
	}
	return words[0], words[1]
}
