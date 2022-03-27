package store

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store links from whom resend to telegram
type Store interface {
	ToTgChatID(ID int64) []int64
	Subscribe(tgChatID, ID int64)
	UnSubscribe(tgChatID, ID int64)
}

type TgVkLinker struct {
	db *mongo.Collection
}

// NewTgVk creates a mongoDB store for linking vk and telegram
func NewTgVk() *TgVkLinker {
	var collection *mongo.Collection
	ctx := context.TODO()

	clientOptions := options.Client().ApplyURI(os.Getenv("MONGODB_URI"))
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	collection = client.Database("go-linker").Collection("VkID-subs")

	return &TgVkLinker{
		db: collection,
	}
}

// auxiliary struct for sending to mongo
type pairID struct {
	VkID int64 `bson:"vk_id"`
	TgID int64 `bson:"tg_id"`
}

// ToTgChatID returns a list of telegram chats which linked with vkChatID
func (t *TgVkLinker) ToTgChatID(vkChatID int64) []int64 {
	var result []int64
	filter := bson.D{{"vk_id", vkChatID}}

	cur, err := t.db.Find(context.TODO(), filter)
	defer cur.Close(context.TODO())

	if err != nil {
		log.Println(err.Error())
		return nil
	}
	for cur.Next(context.TODO()) {
		var elem *pairID
		err := cur.Decode(&elem)
		if err != nil {
			continue
		}
		result = append(result, elem.TgID)
	}

	if err := cur.Err(); err != nil {
		log.Println(err)
		return nil
	}
	return result //[]int64{484251822}
}

// Subscribe adds connection between telegram chat and vk chat
func (t *TgVkLinker) Subscribe(tgChatID, vkChatID int64) {
	countDocuments, err := t.db.CountDocuments(context.TODO(), pairID{VkID: vkChatID, TgID: tgChatID})
	if err != nil {
		log.Println(err.Error())
		return
	}
	if countDocuments != 0 {
		log.Println("VK-> ", vkChatID, " and TG-> ", tgChatID, " was trying to link a lot")
		return
	}
	_, err = t.db.InsertOne(context.TODO(), pairID{VkID: vkChatID, TgID: tgChatID})
	if err != nil {
		log.Println(err.Error())
		return
	}
}

// UnSubscribe remove connection between telegram chat and vk
func (t *TgVkLinker) UnSubscribe(tgChatID, vkChatID int64) {
	_, err := t.db.DeleteOne(context.TODO(), pairID{VkID: vkChatID, TgID: tgChatID})
	if err != nil {
		log.Println(err.Error())
		return
	}
}
