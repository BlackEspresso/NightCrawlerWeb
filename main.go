// main.go
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BlackEspresso/crawlbase"

	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

type Settings struct {
	Listening    string
	Additional   map[string]string
	PhantomPath  string
	MailSettings map[string]string
	S3Buckets    map[string]string
	SMTPPassword string
}

type UserInfo struct {
	Id            string
	UsedRequests  int
	MaxRequests   int
	NextReset     time.Time
	ResetDuration time.Duration
}

type TaskElement struct {
	Id         uuid.UUID
	Func       func(*TaskElement)
	Message    string
	Success    bool
	Param1     string
	Param2     string
	Param3     string
	Additional []interface{}
	ErrorCode  int
}

var appSettings Settings = Settings{}
var ips map[string]int = map[string]int{}
var usedEmails map[string]int = map[string]int{}
var lastFree time.Time = time.Now()
var tasks chan *TaskElement = make(chan *TaskElement, 200)
var accounts gin.Accounts = gin.Accounts{}
var profUsers map[string]*UserInfo = map[string]*UserInfo{}

func main() {
	appSettings = loadSettings()
	profUsers = loadUsers()

	appSettings.PhantomPath = "./"
	appSettings.S3Buckets = map[string]string{
		"Screenshots": "nightcrawlerlinks",
	}
	env := os.Getenv("mailgunpassword")
	appSettings.SMTPPassword = env

	go runQueue()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./static")
	r.GET("/", index)
	r.GET("/screenshot", apiScreenshotPublic)
	r.GET("/siteinfo", apiSiteInfoPublic)
	r.GET("/users/:uid/screenshot", apiScreenshotProfessional)
	r.GET("/pages/:lang/:page", Pages)

	r.GET("crawl/siteinfo", siteinfo)
	r.GET("crawld/screenshot", queueScreenshot)
	r.GET("crawld/siteinfo", siteinfodyn)
	r.GET("crawld/bucketinfo", bucketinfo)
	r.GET("crawld/pageload", siteinfodyn)
	r.GET("crawl/task/add", crawltask)
	r.GET("crawl/task/info", crawltask)
	r.GET("crawl/task/stop", crawltask)
	r.GET("crawl/task/delete", crawltask)

	uname := appSettings.Additional["AdminUserName"]
	pass := appSettings.Additional["AdminPassword"]
	accounts[uname] = pass

	authorized := r.Group("/admin", gin.BasicAuth(accounts))
	authorized.GET("/info", adminInfo)

	r.Run(appSettings.Listening)
}

func newUser() *UserInfo {
	u := UserInfo{}
	u.Id = uuid.NewV4().String()
	u.MaxRequests = 10
	u.ResetDuration = time.Duration(5*24) * time.Hour
	u.NextReset = time.Now().Add(u.ResetDuration)
	return &u
}

func apiSiteInfoPublic(g *gin.Context) {
	clientRequestCount := getClientRequestCount(g)

	if clientRequestCount > 4 {
		res := ErrorResult{"too many request. please wait or buy pro.", 11}
		g.JSON(403, res)
		return
	}

	siteinfo(g)
}

func siteinfo(g *gin.Context) {
	reqUrl := g.Query("url")
	cw := crawlbase.NewCrawler()
	cw.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return nil
	}
	page, err := cw.GetPage(reqUrl, "GET")
	if err != nil {
		g.String(403, err.Error())
		return
	}
	g.JSON(200, page)
}

func loadUsers() map[string]*UserInfo {
	fc, err := ioutil.ReadFile("./professionalusers.json")
	checkFatal(err)
	settings := map[string]*UserInfo{}
	err = json.Unmarshal(fc, &settings)
	checkFatal(err)
	return settings
}

func saveUserInfo() {
	content, err := json.Marshal(profUsers)
	checkFatal(err)
	err = ioutil.WriteFile("./professionalusers.json", content, 0655)
	checkFatal(err)
}

func loadSettings() Settings {
	fc, err := ioutil.ReadFile("settings.json")
	checkFatal(err)
	settings := Settings{}
	err = json.Unmarshal(fc, &settings)
	checkFatal(err)
	return settings
}

func checkFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
