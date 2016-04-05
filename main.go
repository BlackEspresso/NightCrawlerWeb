// main.go
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/BlackEspresso/crawlbase"
	"github.com/gin-gonic/gin"
	"github.com/satori/go.uuid"
)

type Settings struct {
	Listening  string
	Additional map[string]string
}

type UserInfo struct {
	Id            string
	UsedRequests  int
	MaxRequests   int
	NextReset     time.Time
	ResetDuration time.Duration
}

var appSettings Settings = Settings{}
var ips map[string]int = map[string]int{}
var usedEmails map[string]int = map[string]int{}
var lastFree time.Time = time.Now()

var accounts gin.Accounts = gin.Accounts{}
var profUsers map[string]*UserInfo = map[string]*UserInfo{}

func main() {
	appSettings = loadSettings()
	profUsers = loadUsers()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./static")
	r.GET("/screenshot", screenshotPublic)
	r.GET("/siteinfo", siteinfo)
	r.GET("/users/:uid/screenshot", screenshotProfessional)
	r.GET("/", index)
	r.GET("/pages/:lang/:page", Pages)

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
