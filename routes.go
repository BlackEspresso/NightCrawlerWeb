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

func adminInfo(g *gin.Context) {

}

func index(g *gin.Context) {
	lang := g.Request.Header.Get("Accept-Language")
	lang = strings.Split(lang, ";")[0]
	log.Println(lang)
	if strings.HasPrefix(lang, "de") {
		g.Redirect(302, "/de")
	} else {
		g.Redirect(302, "/en")
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
	apiurl, _ := url.Parse("http://localhost:8076/crawld/screenshot")
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

type ScreenshotResult struct {
	Link        string
	DurationSec float64
}

type ErrorResult struct {
	Error     string
	ErrorCode int
}

func screenshotProfessional(g *gin.Context) {
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

func screenshotPublic(g *gin.Context) {
	clientRequestCount, hasKey := ips[g.ClientIP()]
	if !hasKey {
		ips[g.ClientIP()] = 0
	}
	ips[g.ClientIP()] += 1

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

	//reset ip restriction after 1h
	if time.Now().Sub(lastFree).Hours() > 3 {
		lastFree = time.Now()
		ips = map[string]int{}
		usedEmails = map[string]int{}
	}

	// restrict client to 10 requests per hour
	if clientRequestCount > 10 || emailRequestCount > 10 {
		res := ErrorResult{"too many requests from ip or to email, please wait", 11}
		g.JSON(403, res)
		return
	}

	getScreenshot(g)
}

func getLang(key string) map[string]string {
	cRaw, err := ioutil.ReadFile("./templates/lang" + key + ".json")
	if err != nil {
		log.Fatal(err)
	}
	ret := map[string]string{}
	err = json.Unmarshal(cRaw, &ret)
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func indexDE(g *gin.Context) {
	lang := getLang("DE")
	g.HTML(200, "index.tmpl", lang)
}

func indexEN(g *gin.Context) {
	lang := getLang("EN")
	g.HTML(200, "index.tmpl", lang)
}
