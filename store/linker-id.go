package store

import (
	"context"
	"log"
	
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TgVkLinker struct {
	db *mongo.Collection
}

func NewTgVk() *TgVkLinker {
	var collection *mongo.Collection
	ctx := context.TODO()
	
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017/")
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

type pairID struct {
	VkID int64 `bson:"vk_id"`
	TgID int64 `bson:"tg_id"`
}

func (t *TgVkLinker) ToVkChatID(tgChatID int64) []int64 {
	return nil
}

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

func (t *TgVkLinker) MakeSubscribe(tgChatID, vkChatID int64) {
	_, err := t.db.InsertOne(context.TODO(), pairID{VkID: vkChatID, TgID: tgChatID})
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func (t *TgVkLinker) UnSubscribe(tgChatID, vkChatID int64) {
	_, err := t.db.DeleteOne(context.TODO(), pairID{VkID: vkChatID, TgID: tgChatID})
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func remove(l []int64, item int64) []int64 {
	for i, other := range l {
		if other == item {
			return append(l[:i], l[i+1:]...)
		}
	}
	return l
}
