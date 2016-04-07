package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ScreenshotResult struct {
	Link        string
	DurationSec float64
}

type ErrorResult struct {
	Error     string
	ErrorCode int
}

func adminInfo(g *gin.Context) {

}

func getLangFromRequest(g *gin.Context) string {
	lang := g.Request.Header.Get("Accept-Language")
	lang = strings.Split(lang, ";")[0]
	return lang
}

func index(g *gin.Context) {
	lang := getLangFromRequest(g)
	if strings.HasPrefix(lang, "de") {
		g.Redirect(302, "/pages/de/screenshot")
	} else {
		g.Redirect(302, "/pages/en/screenshot")
	}
}

func getScreenshot(g *gin.Context) {
	start := time.Now()

	// build query
	urlQuery := g.Query("url")
	email := g.Query("email")
	format := g.Query("format")

	if urlQuery == "" {
		res := ErrorResult{"url paramter is empty", 3}
		g.JSON(403, res)
		return
	}
	if format == "" && format != "pdf" && format != "jpeg" {
		res := ErrorResult{"format paramter invalid", 4}
		g.JSON(403, res)
		return
	}
	// todo call queue directly
	port := appSettings.Listening
	apiurl, _ := url.Parse("http://localhost" + port + "/crawld/screenshot")
	q := apiurl.Query()
	q.Set("url", urlQuery)
	q.Set("email", email)
	q.Set("format", format)
	apiurl.RawQuery = q.Encode()

	// query api
	res := ScreenshotResult{}
	resp, err := http.Get(apiurl.String())
	if err != nil {
		handleApiError(g, err, 1)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleApiError(g, err, 1)
		return
	}

	dur := time.Now().Sub(start)
	res.Link = string(body)
	res.DurationSec = dur.Seconds()
	g.JSON(200, res)
}

func handleApiError(g *gin.Context, err error, eCode int) {
	errRes := ErrorResult{
		Error:     "internal error",
		ErrorCode: eCode,
	}
	log.Println(err)
	g.JSON(403, errRes)
	return
}

func apiScreenshotProfessional(g *gin.Context) {
	uid := g.Param("uid")
	user, ok := profUsers[uid]
	if !ok {
		res := ErrorResult{"user id not found", 2}
		g.JSON(404, res)
		return
	}

	if time.Now().After(user.NextReset) {
		user.UsedRequests = 0
		user.NextReset = time.Now().Add(user.ResetDuration)
	}

	if user.UsedRequests >= user.MaxRequests {
		res := ErrorResult{"user request quota reached", 10}
		g.JSON(403, res)
		return
	}

	user.UsedRequests += 1

	getScreenshot(g)
}

func getClientRequestCount(g *gin.Context) int {
	clientRequestCount, hasKey := ips[g.ClientIP()]
	if !hasKey {
		ips[g.ClientIP()] = 0
	}
	ips[g.ClientIP()] += 1
	return clientRequestCount
}

func apiScreenshotPublic(g *gin.Context) {
	clientRequestCount := getClientRequestCount(g)

	email := g.Query("email")
	if email == "" {
		g.JSON(403, ErrorResult{"need url parameter", 4})
		return
	}

	emailRequestCount, hasKey := usedEmails[email]
	if !hasKey {
		usedEmails[email] = 0
	}
	usedEmails[email] += 1

	//reset ip restriction after 3 hours
	if time.Now().Sub(lastFree).Hours() > 3 {
		lastFree = time.Now()
		ips = map[string]int{}
		usedEmails = map[string]int{}
	}

	// restrict client to 10 requests per hour
	if clientRequestCount > 4 || emailRequestCount > 4 {
		res := ErrorResult{"too many requests from ip or to email, please wait", 11}
		g.JSON(403, res)
		return
	}

	getScreenshot(g)
}

func getLang(key string) map[string]string {
	key = strings.ToLower(key)
	if key != "de" && key != "en" {
		key = "en"
	}
	cRaw, err := ioutil.ReadFile("./lang/lang.json")
	if err != nil {
		log.Fatal(err)
	}
	ret := map[string]map[string]string{}
	err = json.Unmarshal(cRaw, &ret)
	if err != nil {
		log.Fatal(err)
	}
	return ret[key]
}

func Pages(g *gin.Context) {
	lang := getLang(g.Param("lang"))
	page := g.Param("page")
	page = strings.ToLower(page)
	g.HTML(200, page+".tmpl", lang)
}
