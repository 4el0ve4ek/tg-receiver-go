package receivers

import (
	"context"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/SevereCloud/vksdk/v2/api"
	"github.com/SevereCloud/vksdk/v2/api/params"
	"github.com/SevereCloud/vksdk/v2/events"
	"github.com/SevereCloud/vksdk/v2/longpoll-bot"
	"github.com/SevereCloud/vksdk/v2/object"
	"tg-receiver-bot/model"
	"tg-receiver-bot/store"
)

type IController interface {
	Validate(url string) bool

	Subscribe(url string, tgID int64) (response string)
	Link(url string, tgID int64) (response string)

	UnLink(url string, tgID int64) (response string)
	UnSubscribe(url string, tgID int64) (response string)
}

type linkingValidation struct {
	hash string
	tgID int64
}

type vkBot struct {
	bot               *api.VK
	id                int
	db                *store.TgVkLinker
	waitingAcceptance map[int64]linkingValidation
	messageChan       chan<- *model.TelegramMessage
}

func (v *vkBot) UnLink(url string, tgID int64) string {
	//TODO implement me
	panic("implement me")
}

func (v *vkBot) UnSubscribe(url string, tgID int64) string {
	//TODO implement me
	panic("implement me")
}

func (v *vkBot) Validate(s string) bool {
	matched, _ := regexp.MatchString(`https?://vk\.com/.+`, s)
	return matched
}

func (v *vkBot) Subscribe(url string, tgID int64) string {
	groupId := fetchId(url)
	log.Println(groupId)

	response, err := v.bot.GroupsGetByID(api.Params{"group_id": groupId})

	if err != nil {
		return err.Error()
	}
	if len(response) == 0 || response[0].ID == v.id {
		return "invalid group ID or u try subscribe for sender"
	}
	log.Println(response[0].Name)

	return "Success"
}

func (v *vkBot) Link(url string, tgID int64) string {
	userId := fetchId(url)
	log.Println(userId)

	users, err := v.bot.UsersGet(api.Params{"user_ids": userId})

	if err != nil {
		return err.Error()
	}

	if len(users) == 0 {
		return "invalid user id"
	}
	log.Println(users[0].FirstName + " " + users[0].LastName)

	hashValidator := strconv.FormatInt(rand.Int63(), 10)
	v.waitingAcceptance[int64(users[0].ID)] = linkingValidation{hashValidator, tgID}

	return "Ok. Now write to him () next message \"accept " + hashValidator + "\""
}

// parses url to get group or user id
func fetchId(s string) string {
	re, _ := regexp.Compile(`https?://vk\.com/(.+)`)
	return re.FindStringSubmatch(s)[1]
}

func NewVkBot(messageChan chan<- *model.TelegramMessage, linker *store.TgVkLinker) (bot *vkBot) {
	vk := api.NewVK(os.Getenv("vk_token"))

	group, err := vk.GroupsGetByID(api.Params{})
	if err != nil {
		log.Fatal(err)
	}

	bot = &vkBot{bot: vk, id: group[0].ID,
		db: linker, messageChan: messageChan,
		waitingAcceptance: make(map[int64]linkingValidation)}

	lp, err := longpoll.NewLongPoll(vk, group[0].ID)
	if err != nil {
		log.Fatal(err)
	}
	lp.MessageNew(bot.newMessageReceive)
	go func() {
		if err := lp.Run(); err != nil {
			log.Fatal(err)
		}
	}()

	return
}

func (v *vkBot) newMessageReceive(_ context.Context, obj events.MessageNewObject) {

	b := params.NewMessagesSendBuilder()
	b.Message("Trying to send... check your Telegram")
	b.RandomID(0)
	b.PeerID(obj.Message.PeerID)

	if strings.HasPrefix(obj.Message.Text, "accept") {
		acceptanceWord := strings.SplitN(obj.Message.Text, " ", 2)
		v.acceptLink(int64(obj.Message.PeerID), acceptanceWord[1])
	}
	telegramMessages := parseMessageContaining(obj.Message)

	telegramMessages.ChatIds = v.db.ToTgChatID(int64(obj.Message.PeerID))

	v.messageChan <- telegramMessages

	_, err := v.bot.MessagesSend(b.Params)
	if err != nil {
		log.Fatal(err)
	}
}

func (v *vkBot) acceptLink(vkID int64, acceptance string) {
	var validator linkingValidation
	var ok bool
	if validator, ok = v.waitingAcceptance[vkID]; !ok {
		return
	}
	if validator.hash != acceptance {
		return
	}

	log.Println(vkID)
	log.Println(validator.tgID)

	v.db.MakeSubscribe(validator.tgID, vkID)

	delete(v.waitingAcceptance, vkID)
}

// parses message to model.TelegramMessage
// can convert text, photo, gifs and wall post
func parseMessageContaining(message object.MessagesMessage) (parsed *model.TelegramMessage) {
	parsed = &model.TelegramMessage{}

	if message.Text != "" {
		parsed.AddText(message.Text)
	}

	parsed.AddText("Meme for you: " + message.Text)

	for _, attachment := range message.Attachments {
		switch attachment.Type {
		case "wall":
			parsed.Extend(parseWallPost(attachment.Wall))
		case "photo":
			parsed.AddPhoto(getPhotoUrl(attachment.Photo))
		case "doc":
			if attachment.Doc.Ext == "gif" {
				parsed.AddGif(attachment.Doc.URL)
			} else {
				parsed.AddFile(attachment.Doc.URL)
			}
		}
	}

	return
}

// convert wall post to model.TelegramMessage
// can proceed text, photo, gifs
func parseWallPost(wall object.WallWallpost) (parsed *model.TelegramMessage) {
	parsed = &model.TelegramMessage{}

	if wall.Text != "" {
		parsed.AddText(wall.Text)
	}

	for _, attachment := range wall.Attachments {
		switch attachment.Type {
		case "photo":
			parsed.AddPhoto(getPhotoUrl(attachment.Photo))
		case "doc":
			if attachment.Doc.Ext == "gif" {
				parsed.AddGif(attachment.Doc.URL)
			} else {
				parsed.AddFile(attachment.Doc.URL)
			}
		}
	}

	return
}

// utility to get last url with photo
func getPhotoUrl(photo object.PhotosPhoto) string {
	photoSizes := photo.Sizes
	if len(photoSizes) == 0 {
		return ""
	}
	photoUrl := photoSizes[len(photoSizes)-1].URL
	return photoUrl
}
