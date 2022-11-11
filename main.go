/**
* @Author: Li Feiyang
* @Date: 2022/11/9 10:57
 */

package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
	"net/http"
	"strconv"
	"time"
)

const Database = "pastebin"

//var languageList = []string{"C", "C++", "Java", "Python", "Go", "JavaScript"}

/**
 * 连接数据库。
 * 值得注意的是调用了 DataInit(client) 和 VerifyInit(client)，这是MongoDB TTL功能的要求：开启计时器以支持自动删除功能。
 * @return : *mongo.Client
 */
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
		logrus.Fatal("FAILED to connect MongoDB!")
	} else {
		logrus.Info("Successfully connected MongoDB and pinged.")
	}

	DataInit(client)
	VerifyInit(client)

	return client
}

/**
 * 为服务提供各种路由方法。
 */
func setupRouter(client *mongo.Client) *gin.Engine {
	r := gin.Default()

	r.GET("/pastebin/verify", func(c *gin.Context) {
		//param := c.Request.URL.RawQuery
	})

	/**
	 * POST方法，接收包含密码的表单。查数据库进行比对，如果通过就设置一条cookie，
	 * 向数据库中添加临时条目（过期时间为 30 分钟），内容为用户的 sessionId 和该 sessionId 已被授权访问的网页地址。
	 */
	r.POST("/pastebin/verify", func(c *gin.Context) {
		password := c.PostForm("password")
		paramUrl := c.PostForm("url")
		/*param := c.Request.URL.RawQuery
		result := strings.Split(param, "=")
		paramUrl := result[len(result)-1]*/
		logrus.Info("Verify url + " + paramUrl + " with password: " + password)
		if passwordVerify(client, password, paramUrl) == true {
			// 密码正确
			sessionID, err := c.Cookie("sessionID")
			if err != nil {
				sessionID = generateUrl()
				logrus.Info("Rand sessionID generated:" + sessionID)
				c.SetCookie("sessionID", sessionID, 1800, "/", "localhost", false, true)
			}
			//c.String(http.StatusOK, "VERIFY SUCCESS!")
			InsertVerify(client, sessionID, paramUrl)
			logrus.Info("Verify succeeded, redirect to: " + paramUrl)
			c.Redirect(http.StatusFound, "/pastebin/"+paramUrl)
		} else {
			//密码错误
			//c.String(http.StatusOK, "VERIFY FAIL!")
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 10003,
				},
			})
		}
	})

	/**
	 * POST方法，接收包含文件数据的表单。
	 * 初始化文件数据结构File，然后对数据进行校验，若通过则加入数据库
	 *
	 */
	r.POST("/pastebin/file", func(c *gin.Context) {
		logrus.Info("POST： submit file data")
		var fileStruct = new(File)
		// 允许访问的次数（默认1
		times := c.DefaultPostForm("times", "1")
		fileStruct.Times, _ = strconv.Atoi(times)
		// 到期自动删除的时间（默认3600s = 60min
		expire := c.DefaultPostForm("expire", "3600")
		intExpire, _ := strconv.Atoi(expire)
		// 设置MongoDB的TTL
		fileStruct.Timestamp = time.Now()
		fileStruct.CreatedAt = time.Now().Add(time.Second * time.Duration(intExpire))
		// 密码和Url
		fileStruct.Password = c.PostForm("password")
		fileStruct.Url = generateUrl()
		logrus.Info("Rnd url generated：" + fileStruct.Url)
		var result bool
		//var err error
		// 获取上传的文件头
		file, fileHeader, err := c.Request.FormFile("data")
		if err != nil {
			logrus.Error("Submit file: unknown error!" + err.Error())
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 10003,
				},
			})
		} else {
			fileStruct.Name = fileHeader.Filename
			//处理文件数据并校验
			fileStruct.Data, result, fileStruct.Category = getFileData(c, file, fileHeader)
			if result == true { // 通过校验
				err2 := installFile(client, *fileStruct)
				if err2 != nil {
					c.JSON(http.StatusOK, gin.H{
						"message": "POST",
						"code":    0,
						"data": gin.H{
							"status": 10003,
						},
					})
				} else { //不通过校验
					c.JSON(http.StatusOK, gin.H{
						"message": "POST",
						"code":    0,
						"data": gin.H{
							"status": 0,
							"url":    fileStruct.Url,
						},
					})
				}
			}
		}

	})

	/**
	 * POST方法，接收包含上传的代码的表单。
	 *
	 */
	r.POST("/pastebin/submit", func(c *gin.Context) {
		logrus.Info("POST: submit text")
		var file = new(File)
		// 下载次数（默认1
		times := c.DefaultPostForm("times", "1")
		file.Times, _ = strconv.Atoi(times)
		// 过期时间（默认3600s = 60min
		expire := c.DefaultPostForm("expire", "3600")
		intExpire, _ := strconv.Atoi(expire)
		// 密码
		file.Password = c.PostForm("password")
		file.Url = generateUrl()
		logrus.Info("Url generated : " + file.Url)
		// MongoDB TTL 时间戳
		file.Timestamp = time.Now()
		file.CreatedAt = time.Now().Add(time.Second * time.Duration(intExpire))

		if c.PostForm("highlight") == "true" {
			file.Highlight = true
		} else {
			file.Highlight = false
		}

		file.Text = c.PostForm("text")
		file.Language = c.PostForm("language")
		//上传加入数据库并处理错误
		err := installFile(client, *file)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 10001,
					"url":    "",
				},
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "POST",
				"code":    0,
				"data": gin.H{
					"status": 0,
					"url":    file.Url,
				},
			})
		}
	})

	/**
	 * 试图访问获取资源。首先检查是否有有效的 SessionID，若没有则直接跳转到验证页面，
	 *	否则检查数据库是否能查询到数据，视情况返回数据。
	 */
	r.GET("/pastebin/:path", func(c *gin.Context) {
		//c.Header("Content-Type", "text/markdown")
		path := c.Param("path")
		logrus.Info("GET: get the data of url: " + path)
		sessionID, _ := c.Cookie("sessionID")
		// 检查有无有效的sessionID
		err, result := verifySessionID(client, sessionID, path)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{ // TODO: API update
				"message": "GET",
				"code":    0,
				"data": gin.H{
					"status": 10001, // Session 验证出现问题
				},
			})
		} else if result == true {
			// 如果有 SessionID， 向前端查询数据，视情况是否返回文件
			path := c.Param("path")
			e, result, FILE := queryUrl(client, path)
			if e != nil {
				c.JSON(http.StatusOK, gin.H{ // TODO: API update
					"message": "GET",
					"code":    0,
					"data": gin.H{
						"status": 10001, // 查询数据过程出现问题
					},
				})
			} else if result && FILE.Times > 0 {

				switch FILE.Category {
				case "txt":
					c.Header("Content-Type", "text/plain")
				case "md":
					c.Header("Content-Type", "text/markdown")
				case "csv":
					c.Header("Content-Type", "text/csv")
				case "tex":
					c.Header("Content-Type", "text/x-tex")
				default:
					c.Header("Content-Type", "text/plain")
				}
				returnData(client, c)
			} else {
				// result == false
				c.JSON(http.StatusOK, gin.H{ // TODO: API update
					"message": "GET",
					"code":    0,
					"data": gin.H{
						"status": 10001, // 获取不到文件（数据库找不到文件）
					},
				})
			}
		} else {
			// 没有session 则跳转到验证
			logrus.Info("Redirect to verify Page!")
			c.Redirect(http.StatusMovedPermanently, "/pastebin/verify?url="+path)
		}
	})
	return r
}

func main() {

	logrus.SetLevel(logrus.TraceLevel)
	client := con()
	r := setupRouter(client)
	err := r.Run(":8080")
	if err != nil {
		logrus.Fatal("ERROR in Run client in port 8080！")
	}

}
