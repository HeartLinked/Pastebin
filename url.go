package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"math/rand"
	"time"
)

var byteString []byte = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func init() {
	// 保证每次生成的随机数不一样
	rand.Seed(time.Now().UnixNano())
}

func RandStr(n int) string {
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = byteString[rand.Int31()%62]
	}
	return string(result)
}

func Generateurl() string {
	for {
		length := rand.Intn(25)
		if length > 15 {
			return RandStr(length)
		}
	}
}

func Queryurl(client *mongo.Client, s string) (int, File) {
	collection := client.Database(Database).Collection("data")
	result := File{}
	err := collection.FindOne(context.TODO(), bson.D{{"url", s}}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			fmt.Println("sdasdsa")
			return 0, result
		}
		panic(err)
	}
	return 1, result
}

func Updateurl(client *mongo.Client, s string) (int, File) {
	collection := client.Database(Database).Collection("data")
	filter := bson.D{{"url", s}}
	result := File{}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	before := result.Times
	logrus.SetLevel(logrus.TraceLevel)
	logrus.Trace("trace msg")
	if before <= 1 {
		collection.DeleteMany(context.TODO(), filter)
	} else {
		collection.UpdateOne(context.TODO(), filter, bson.D{{"$set", bson.D{{"times", before - 1}}}})
	}
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			fmt.Println("sdasdsa")
			return 0, result
		}
		panic(err)
	}
	return 1, result
}
