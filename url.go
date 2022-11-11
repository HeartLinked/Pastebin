package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"math/rand"
	"time"
)

// 随机串字符范围
var byteString = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func init() {
	// 保证每次生成的随机数不一样
	rand.Seed(time.Now().UnixNano())
}

// 生成一个长度为m的随机字符串
func randStr(n int) string {
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
			url := randStr(length)
			return url
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
func queryUrl(client *mongo.Client, s string) (error, bool, File) {
	collection := client.Database(Database).Collection("data")
	result := File{}
	err := collection.FindOne(context.TODO(), bson.D{{"url", s}}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			logrus.Info("Query data from url:" + s + ", result: FAIL")
			return nil, false, result
		} else {
			logrus.Error("ERROR in query data using url: " + err.Error())
			return err, false, result
		}
	}
	logrus.Info("Query data from url:" + s + ", result: SUCCESS")
	return nil, true, result
}

/**
 * 在访问数据以后，对数据库的数据进行更新。减少剩余的可访问次数。如果剩余次数小于等于 0 则删除该条数据。
 *
 * @param client : *mongo.Client
 * @param url : 要更新的数据的 url
 */
func updateData(client *mongo.Client, url string) {
	collection := client.Database(Database).Collection("data")
	filter := bson.D{{"url", url}}
	result := File{}
	_ = collection.FindOne(context.TODO(), filter).Decode(&result)
	before := result.Times
	if before <= 1 {
		logrus.Info("The times of visits remaining is 0.")
		_, err := collection.DeleteMany(context.TODO(), filter)
		if err != nil {
			logrus.Error("Failed to delete data from database!")
		} else {
			logrus.Info("Updated data to set times-- successfully!")
		}
	} else {
		_, err := collection.UpdateOne(context.TODO(), filter, bson.D{{"$set", bson.D{{"times", before - 1}}}})
		if err != nil {
			logrus.Error("Failed to reset data from database!")
		} else {
			logrus.Info("Updated data to set times-- successfully!")
		}
	}
}

/**
 * 验证 SessionID 对于这个 url 是否有效。
 *
 * @param s : SessionID
 * @param url : 要访问的 url：去数据库查 Session 对应的 url 列表有没有这一项
 * @return error: 验证过程中是否出现了错误，如果出错返回交由上层处理
 * @return bool: 验证结果是否成功 true 成功, false 失败
 */
func verifySessionID(client *mongo.Client, s string, url string) (error, bool) {
	collection := client.Database(Database).Collection("verify")
	filter := bson.D{{"sessionID", s}}
	result := Verify{}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			logrus.Info("Verify SessionID:" + s + "with url:" + url + ", Result: FAIL")
			return nil, false
		} else {
			logrus.Error("ERROR to verify sessionID :" + err.Error())
			return err, false
		}
	}
	for _, Url := range result.Url {
		if Url == url {
			logrus.Info("Verify SessionID:" + s + "with url:" + url + ", Result: SUCCESS")
			return nil, true
		}
	}
	logrus.Info("Verify SessionID:" + s + "with url:" + url + ", Result: FAIL")
	return nil, false
}
