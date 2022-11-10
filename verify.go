/**
* @Author: Li Feiyang
* @Date: 2022/11/9 15:48
 */

package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type Verify struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"createdAt,omitempty"`
	Timestamp time.Time          `bson:"timestamp,omitempty"`

	SessionID string   `bson:"sessionID" json:"sessionID"`
	Url       []string `bson:"url" json:"url"`
}

/**
 * 对于存储的密码数据初始化MongoDB的TTL计时器，以支持到期自动删除
 *
 * @param client : *mongo.Client
 */

func VerifyInit(client *mongo.Client) {
	collection := client.Database(Database).Collection("verify")
	model := mongo.IndexModel{
		Keys:    bson.M{"createdAt": 1},
		Options: options.Index().SetExpireAfterSeconds(30 * 60),
	}
	_, err := collection.Indexes().CreateOne(context.TODO(), model)
	if err != nil {
		log.Fatal(err)
	}
}

/**
 * 对于输入的密码，查数据库检查密码是否正确
 *
 * @param s : 输入的密码串
 * @param url : 查询的url
 * @return : 密码正确 true / 密码错误 false
 */
func passwordVerify(client *mongo.Client, s string, url string) bool {
	i, file := queryUrl(client, url)
	if i == false && file.Password == s {
		return true
	}
	return false
}

/**
 * 向数据库插入一条SessionID记录，持续时间30min，到期自动删除
 *
 * @param client : *mongo.Client
 * @param sessionID : 记录字符串
 * @param url : url string
 * @return :
 */

func InsertVerify(client *mongo.Client, sessionID string, url string) {
	collection := client.Database(Database).Collection("verify")
	result := Verify{}
	filter := bson.D{{"sessionID", sessionID}}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			var t = make([]string, 1)
			t[0] = url
			result = Verify{
				ID:        primitive.ObjectID{},
				CreatedAt: time.Now(),
				Timestamp: time.Now(),
				SessionID: sessionID,
				Url:       t,
			}
			_, err := collection.InsertOne(context.TODO(), result)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
	} else {
		t := result.Url
		t = append(t, url)
		_, err := collection.UpdateOne(context.TODO(), filter, bson.D{{"$set", bson.D{{"url", t}}}})
		if err != nil {
			log.Fatal(err)
		}
	}
}
