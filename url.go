package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"math/rand"
	"time"
)

// 随机串字符范围
var byteString = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func init() {
	// 保证每次生成的随机数不一样
	rand.Seed(time.Now().UnixNano())
}

func RandStr(n int) string { // 生成一个长度为m的随机字符串
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = byteString[rand.Int31()%62]
	}
	return string(result)
}

/**
 * 随机生成一条Url：长度[16, 25]
 *
 * @return : 生成的Url
 */
func generateUrl() string {
	for {
		length := rand.Intn(25)
		if length > 15 {
			return RandStr(length)
		}
	}
}

/**
 * 从指定的Url查询数据是否存在。如果存在数据，返回true，否则返回false
 *
 * @param client : *mongo.Client
 * @param s : Url string
 * @return bool : 存在数据true / 不存在数据 false
 * @return File : 数据文件结构体
 */
func queryUrl(client *mongo.Client, s string) (bool, File) {
	collection := client.Database(Database).Collection("data")
	result := File{}
	err := collection.FindOne(context.TODO(), bson.D{{"url", s}}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return false, result
		}
		panic(err)
	}
	return true, result
}

func updateUrl(client *mongo.Client, s string) (int, File) {
	collection := client.Database(Database).Collection("data")
	filter := bson.D{{"url", s}}
	result := File{}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	before := result.Times
	logrus.SetLevel(logrus.TraceLevel)
	logrus.Trace("trace msg")
	if before <= 1 {
		_, err := collection.DeleteMany(context.TODO(), filter)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		_, err := collection.UpdateOne(context.TODO(), filter, bson.D{{"$set", bson.D{{"times", before - 1}}}})
		if err != nil {
			log.Fatal(err)
		}
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return 0, result
		}
		panic(err)
	}
	return 1, result
}

func VerifySessionID(client *mongo.Client, s string, url string) bool {
	collection := client.Database(Database).Collection("verify")
	filter := bson.D{{"sessionID", s}}
	result := Verify{}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return false
		}
		panic(err)
	}
	for _, Url := range result.Url {
		if Url == url {
			return true
		}
	}
	return false
}
