/**
* @Author: Li Feiyang
* @Date: 2022/11/9 11:24
 */

package main

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"io/fs"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const FILESIZE = 20971520

var suffix = []string{"txt", "md", "tex", "csv"}

// 代码和文件的存储数据结构

type File struct {
	// 主键
	ID primitive.ObjectID `bson:"_id,omitempty"`
	// 时间戳，用于支持 MongoDB 的 TTL
	CreatedAt time.Time `bson:"createdAt,omitempty"`
	Timestamp time.Time `bson:"timestamp,omitempty"`
	Url       string    `bson:"url,omitempty" json:"url"`
	Password  string    `bson:"password,omitempty" json:"password,omitempty"`
	// 剩余访问次数
	Times int    `bson:"times,omitempty" json:"times"`
	Data  []byte `bson:"data,omitempty" json:"data"`

	// 对于文件：文件名，后缀名
	Name     string `bson:"name,omitempty" json:"name,omitempty"`
	Category string `bson:"category,omitempty" json:"category,omitempty"`

	// 对于代码：是否高亮，语言类型，文本内容
	Highlight bool   `bson:"highlight,omitempty" json:"highlight,omitempty"`
	Language  string `bson:"language,omitempty" json:"language,omitempty"`
	Text      string `bson:"text,omitempty" json:"text,omitempty"`
}

/**
 * 对于存储的数据（包括文件和文本）初始化MongoDB的TTL计时器，以支持到期自动删除
 *
 * @param client : *mongo.Client
 */

func DataInit(client *mongo.Client) {
	logrus.Info("Initialize data TTL function...")
	collection := client.Database(Database).Collection("data")
	model := mongo.IndexModel{
		Keys:    bson.M{"createdAt": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	_, err := collection.Indexes().CreateOne(context.TODO(), model)
	if err != nil {
		logrus.Error(err)
		logrus.Error("Failed to initialize the TTL function of MongoDB!")
	} else {
		logrus.Info("The TTL function of MongoDB was opened successfully!")
	}
}

/**
 * 将上传的文件写入数据库
 *
 * @param a : client *mongo.Client
 * @param b : 保存文件的结构体
 * @return : error : 写入数据库错误
 */
func installFile(client *mongo.Client, file File) error {
	logrus.Info("Write data into Database..")
	collection := client.Database(Database).Collection("data")
	_, err := collection.InsertOne(context.TODO(), file)
	if err != nil {
		logrus.Error("ERROR in writing data into Database")
	} else {
		logrus.Info("Wrote data into database successfully!")
	}
	return err
}

/**
 * 文件大小校验： 不应超过FILESIZE（默认20MB
 * 文件后缀名校验： 文件后缀名在给定字符串数组中取值，默认 suffix = []string{"txt", "md", "tex", "csv"}
 *
 * @param a :
 * @param b :
 * @return bool: 文件通过校验true， 不通过false
 */
func checkFile(c *gin.Context, fileHeader *multipart.FileHeader) (bool, string) {
	if fileHeader.Size > FILESIZE {
		logrus.Info("Check File Failed: file size exceeds the maximum range")
		c.JSON(http.StatusOK, gin.H{
			"message": "POST",
			"code":    0,
			"data": gin.H{
				"status": 10002,
			},
		})
		return false, ""
	} else {
		name := fileHeader.Filename
		split := strings.Split(name, ".")
		fileSuffix := split[len(split)-1]
		check := false
		for i := 0; i < len(suffix); i++ {
			if fileSuffix == suffix[i] {
				check = true
				break
			}
		}
		if check != true { // 校验失败
			logrus.Info("Check File Failed: file suffix is illegal")
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 10001,
				},
			})
			return false, fileSuffix
		}
		logrus.Info("Check File Succeeded!")
		return true, fileSuffix
	}
}

/**
 * 处理文件数据，返回文件字节流数据、校验结果、后缀名
 *
 * @param c : *gin.Context
 * @param file : 文件处理的接口
 * @param fileHeader : 文件处理头
 * @return b: 文件数据字节流
 * @return result: 文件校验结果
 * @return fileSuffix: 文件后缀名
 */
func getFileData(c *gin.Context, file multipart.File, fileHeader *multipart.FileHeader) (b []byte, result bool, fileSuffix string) {
	// 文件大小、后缀名校验
	result, fileSuffix = checkFile(c, fileHeader)
	if result == false {
		return nil, false, ""
	}
	// 将上传的文件类型转成字节流
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, false, fileSuffix
	}
	logrus.Info("Got file data successfully")
	return buf.Bytes(), true, fileSuffix
}

/**
 * 如果在数据库中查到数据，向前端返回数据。
 *
 * @param client : *mongo
 * @param c : *gin.Context
 * @return :
 */
func returnData(client *mongo.Client, c *gin.Context) {
	path := c.Param("path")
	_, _, FILE := queryUrl(client, path)
	updateData(client, path)
	// 如果是文件类型
	if FILE.Text == "" {
		permissions := 0777 // or whatever you need
		err := ioutil.WriteFile("file", FILE.Data, fs.FileMode(permissions))
		if err != nil {
			//服务器写文件出现错误：500
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "GET",
				"code":    0,
				"data": gin.H{
					"status": 10001,
				},
			})
			logrus.Error("ERROR in writing data to database: " + err.Error())
		}
		c.FileAttachment("file", FILE.Name)
	} else {
		// 如果是代码类型
		logrus.Info("Return Data: (plain text, highlight =" + strconv.FormatBool(FILE.Highlight) + ")")
		c.JSON(http.StatusOK, gin.H{
			"message": "GET",
			"code":    0,
			"data": gin.H{
				"highlight": FILE.Highlight,
				"text":      FILE.Text,
			},
		})
	}

}
