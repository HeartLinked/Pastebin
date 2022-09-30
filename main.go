package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const Database = "pastebin"

var suffix = []string{"txt", "md", "tex", "csv"}

type File struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"createdAt,omitempty"`
	Timestamp time.Time          `bson:"timestamp,omitempty"`

	Data     []byte `bson:"data" json:"data"`
	Name     string `bson:"name" json:"name"`
	Url      string `bson:"url" json:"url"`
	Password string `bson:"password" json:"password,omitempty"`
	Times    int    `bson:"times" json:"times"`
}

type Verify struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"createdAt,omitempty"`
	Timestamp time.Time          `bson:"timestamp,omitempty"`

	SessionID string   `bson:"sessionID" json:"sessionID"`
	Url       []string `bson:"url" json:"url"`
}

func DataInit(client *mongo.Client) {
	collection := client.Database(Database).Collection("data")
	model := mongo.IndexModel{
		Keys:    bson.M{"createdAt": 1},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	_, err := collection.Indexes().CreateOne(context.TODO(), model)
	if err != nil {
		log.Fatal(err)
	}
}

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

func Installfile(client *mongo.Client, file File) {
	collection := client.Database(Database).Collection("data")
	one, err := collection.InsertOne(context.TODO(), file)
	if err != nil {
		panic(err)
	}
	fmt.Println(one)
}

func Uploadfile(c *gin.Context) (b []byte, s string, flag bool, e error) {
	file, fileheader, err := c.Request.FormFile("data")
	name := fileheader.Filename
	size := fileheader.Size
	flag = false
	if size > 20971520 {
		c.JSON(http.StatusOK, gin.H{
			"message": "POST",
			"code":    0,
			"data": gin.H{
				"status": 10002,
			},
		})
		flag = true
	} else {
		cpyname := name
		result := strings.Split(cpyname, ".")
		filesuffix := result[len(result)-1]
		var check = false
		for i := 0; i < len(suffix); i++ {
			if filesuffix == suffix[i] {
				check = true
			}
		}
		if check != true {
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 10001,
				},
			})
			flag = true
		}
	}
	if err != nil {
		return nil, name, flag, err
	}
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, name, flag, err
	}
	return buf.Bytes(), name, flag, err
}

func Passwordverify(cilent *mongo.Client, s string, url string) bool {
	i, file := Queryurl(cilent, url)
	if i == 1 && file.Password == s {
		return true
	}
	return false
}

func InsertVerify(client *mongo.Client, sessionID string, url string) {
	collection := client.Database(Database).Collection("verify")
	result := Verify{}
	filter := bson.D{{"sessionID", sessionID}}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			var t = make([]string, 1024)
			t[0] = url
			result = Verify{
				ID:        primitive.ObjectID{},
				CreatedAt: time.Time{},
				Timestamp: time.Time{},
				SessionID: sessionID,
				Url:       t,
			}
			_, err := collection.InsertOne(context.TODO(), result)
			if err != nil {
				panic(err)
			}
		} else {
			panic(err)
		}
	} else {
		t := result.Url
		t[len(t)] = url
		_, err := collection.UpdateOne(context.TODO(), filter, bson.D{{"$set", bson.D{{"url", t}}}})
		if err != nil {
			log.Fatal(err)
		}
	}
}

func con() *mongo.Client {
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI("mongodb+srv://xfydemx:LFYmdb1213-@cluster0.ivvl0ib.mongodb.net/?retryWrites=true&w=majority").
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	fmt.Println("Successfully connected and pinged.")

	DataInit(client)
	VerifyInit(client)

	return client
}

func setupRouter(client *mongo.Client) *gin.Engine {
	r := gin.Default()

	r.GET("/pastebin/verify", func(c *gin.Context) {
		//param := c.Request.URL.RawQuery
		fmt.Println()
	})
	r.POST("/pastebin/verify", func(c *gin.Context) {
		password := c.PostForm("password")
		param := c.Request.URL.RawQuery
		result := strings.Split(param, "=")
		paramurl := result[len(result)-1]
		fmt.Println(password)
		fmt.Println(paramurl)
		if Passwordverify(client, password, paramurl) == true {
			sessionID := Generateurl()
			c.SetCookie("sessionID", sessionID, 1800, "/", "localhost", false, true)
			c.String(http.StatusOK, "VERIFY SUCCESS!")
			InsertVerify(client, sessionID, paramurl)
			c.Redirect(http.StatusMovedPermanently, "/pastebin/"+paramurl)
		} else {
			c.String(http.StatusOK, "VERIFY FAIL!")
		}
	})

	r.POST("/pastebin/file", func(c *gin.Context) {
		var file = new(File)
		times := c.DefaultPostForm("times", "1")
		file.Times, _ = strconv.Atoi(times)
		expire := c.DefaultPostForm("expire", "3600")
		intExpire, _ := strconv.Atoi(expire)
		file.Password = c.PostForm("password")
		file.Url = Generateurl()
		file.Timestamp = time.Now()
		file.CreatedAt = time.Now().Add(time.Second * time.Duration(intExpire))
		var flag bool
		var err error
		file.Data, file.Name, flag, err = Uploadfile(c)
		if flag == false && err == nil {
			Installfile(client, *file)
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 0,
					"url":    file.Url,
				},
			})
		} else if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 10003,
				},
			})
		}
	})

	r.GET("/pastebin/:path", func(c *gin.Context) {
		path := c.Param("path")
		// TODO: session
		status, FILE := Queryurl(client, path)
		if status == 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "GET",
				"code":    0,
				"data":    "",
			})
		} else if FILE.Times <= 0 {
			c.JSON(http.StatusNotFound, gin.H{
				"message": "GET",
				"code":    0,
				"data":    "",
			})
		} else if FILE.Password != "" {
			// TODO: session
			c.Redirect(http.StatusMovedPermanently, "/pastebin/verify?url="+path)

		} else {
			Updateurl(client, path)
			permissions := 0777 // or whatever you need
			err := ioutil.WriteFile("file", FILE.Data, fs.FileMode(permissions))
			if err != nil {
				// handle error
			}
			c.FileAttachment("file", FILE.Name)
		}
	})
	return r
}

func main() {
	client := con()

	r := setupRouter(client)
	err := r.Run(":8080")
	if err != nil {
		log.Fatal()
	}

}
