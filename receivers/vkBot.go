package receivers

import (
	"context"
	"errors"
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

type Controller interface {
	Validate(url string) bool

	Link(url string, tgID int64) (response string)

	UnLink(url string, tgID int64) (response string)
}

type linkingValidation struct {
	hash string
	tgID int64
}

type vkBot struct {
	apiVK             *api.VK
	db                store.Store
	waitingAcceptance map[int64]linkingValidation
	messageChan       chan<- *model.TelegramMessage
}

// UnLink tg chat from vk user
func (v *vkBot) UnLink(url string, tgID int64) string {
	userId := fetchVKID(url)
	log.Println("Trying unlink to ", userId)

	user, err := v.fetchUser(userId)

	if err != nil {
		return err.Error()
	}
	v.db.UnSubscribe(tgID, int64(user.ID))

	return "Done"
}

// Validate if url is vk link
func (v *vkBot) Validate(s string) bool {
	matched, _ := regexp.MatchString(`https?://vk\.com/.+`, s)
	return matched
}

// Link vk user and tg chat
func (v *vkBot) Link(url string, tgID int64) string {
	userId := fetchVKID(url)
	log.Println("Trying link to ", userId)

	user, err := v.fetchUser(userId)

	if err != nil {
		return err.Error()
	}

	hashValidator := strconv.FormatInt(rand.Int63(), 10)
	v.waitingAcceptance[int64(user.ID)] = linkingValidation{hashValidator, tgID}

	return "Ok. Now write to him () next message \"accept " + hashValidator + "\""
}

// parses url to get group or user id
func fetchVKID(s string) string {
	re, _ := regexp.Compile(`https?://vk\.com/(.+)`)
	return re.FindStringSubmatch(s)[1]
}

func NewVkBot(messageChan chan<- *model.TelegramMessage, linker store.Store) (bot *vkBot) {
	vk := api.NewVK(os.Getenv("vk_token"))

	group, err := vk.GroupsGetByID(api.Params{})
	if err != nil {
		log.Fatal(err)
	}

	bot = &vkBot{
		apiVK:             vk,
		db:                linker,
		messageChan:       messageChan,
		waitingAcceptance: make(map[int64]linkingValidation),
	}

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

// validates all incoming messages
func (v *vkBot) newMessageReceive(_ context.Context, obj events.MessageNewObject) {
	log.Println("VK message from ", obj.Message.PeerID, ". MID: ", obj.Message.ID)
	if strings.HasPrefix(obj.Message.Text, "accept") {
		acceptanceWord := strings.SplitN(obj.Message.Text, " ", 2)
		v.acceptLink(int64(obj.Message.PeerID), acceptanceWord[1])
		return
	}

	messagesToTG := parseMessageContaining(obj.Message)

	messagesToTG.ChatIds = v.db.ToTgChatID(int64(obj.Message.PeerID))

	v.messageChan <- messagesToTG

}

// accepts linking of vk and tg chats
func (v *vkBot) acceptLink(vkID int64, acceptance string) {
	var validator linkingValidation
	var ok bool

	b := params.NewMessagesSendBuilder()
	b.RandomID(0)
	b.PeerID(int(vkID))

	if validator, ok = v.waitingAcceptance[vkID]; !ok {
		b.Message("Nobody to accept.")
	} else if validator.hash != acceptance {
		b.Message("Invalid hash word.")
	} else {
		log.Println(vkID, " linked with ", validator.tgID)
		v.db.Subscribe(validator.tgID, vkID)
		delete(v.waitingAcceptance, vkID)
		b.Message("Successfully linked!")
	}

	_, err := v.apiVK.MessagesSend(b.Params)
	if err != nil {
		log.Println(err)
	}
}

// auxiliary function to get vk user by its id
func (v *vkBot) fetchUser(id string) (*object.UsersUser, error) {
	users, err := v.apiVK.UsersGet(api.Params{"user_ids": id})

	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, errors.New("invalid user id")
	}

	return &users[0], nil
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
