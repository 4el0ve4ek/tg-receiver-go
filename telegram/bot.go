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
	controller   []receivers.IController
	messagesChan chan *model.TelegramMessage
}

// New generate tg bot and all bot-receivers for him
func New() *Bot {
	messageChan := make(chan *model.TelegramMessage)
	VkController := receivers.NewVkBot(messageChan, store.NewTgVk())

	return &Bot{
		controller:   []receivers.IController{VkController},
		messagesChan: messageChan,
	}
}

func (b *Bot) Start() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("tg_token"))
	if err != nil {
		panic(err)
	}

	//bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	go b.observe(bot)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		text := update.Message.Text
		command, link := separate(text)

		chatID := update.Message.Chat.ID

		var curController receivers.IController

		for _, controller := range b.controller {
			if controller.Validate(link) {
				curController = controller
			}
		}
		if curController == nil {
			bot.Send(tgbotapi.NewMessage(chatID, "invalid link"))
			continue
		}
		switch command {
		case "/link":
			response := curController.Link(link, chatID)
			bot.Send(tgbotapi.NewMessage(chatID, response))
		}

	}
}

// waits for messages and send them to TG
func (b *Bot) observe(bot *tgbotapi.BotAPI) {
	for {
		select {
		case msg := <-b.messagesChan:
			for _, tgMsg := range ConvertModel(msg) {
				_, err := bot.Send(tgMsg)
				if err != nil {
					log.Print(err)
				}
			}
		}
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

//allowCommands contains valid command
var allowCommands = []string{"/sub", "/unsub", "/link", "unlink"}

// check that command is one of allowed
func isCommandValid(command string) bool {
	for _, com := range allowCommands {
		if com == command {
			return true
		}
	}
	return false
}

// splits the message for command part and link part
func separate(text string) (command string, link string) {
	words := strings.SplitN(text, " ", 2)
	if len(words) < 2 {
		return words[0], ""
	}
	return words[0], words[1]
}
