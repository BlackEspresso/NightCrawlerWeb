// main.go
package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./static")
	r.GET("/screenshot", screenshot)
	r.GET("/", index)
	r.GET("/de", indexDE)
	r.GET("/en", indexEN)
	if gin.IsDebugging() {
		r.Run(":8080")
	} else {
		r.Run(":80")
	}

}

var ips map[string]int = map[string]int{}
var lastFree time.Time = time.Now()

func screenshot(g *gin.Context) {
	urlQuery := g.Query("url")

	clientRequestCount, hasKey := ips[g.ClientIP()]
	if !hasKey {
		ips[g.ClientIP()] = 0
	}
	ips[g.ClientIP()] += 1

	log.Println(g.ClientIP(), clientRequestCount)

	//reset ip restriction after 1h
	if time.Now().Sub(lastFree).Hours() > 1 {
		lastFree = time.Now()
		ips = map[string]int{}
	}

	// restrict client to 10 requests per hour
	if clientRequestCount > 10 {
		g.String(403, "too many requests from ip "+g.ClientIP()+", please wait")
		return
	}

	apiurl, _ := url.Parse("http://localhost:8076/crawld/screenshot")
	q := apiurl.Query()
	q.Set("url", urlQuery)
	apiurl.RawQuery = q.Encode()

	res, err := http.Get(apiurl.String())
	if err != nil {
		log.Println(err)
		g.String(403, "error, see logs")
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println(err)
		g.String(403, "error, see logs")
		return
	}
	g.String(res.StatusCode, string(body))
}

func index(g *gin.Context) {
	lang := g.Request.Header.Get("Accept-Language")
	lang = strings.Split(lang, ";")[0]
	log.Println(lang)
	if strings.HasPrefix(lang, "de") {
		g.Redirect(301, "/de")
	} else {
		g.Redirect(301, "/en")
	}
}

func indexDE(g *gin.Context) {
	g.HTML(200, "index.tmpl", gin.H{
		"title":             "Screenshot Website",
		"needmoresnaphosts": "Benötigen Sie mehr Screenshots oder eine API?",
		"gopro1":            "Wechseln Sie zu Websnapshot Professional für",
		"gopro2":            "1000 Screenshots pro Monat & unbegrenztes teilen der Links",
		"anyquestions":      "noch Fragen? schreiben Sie uns",
		"emailaddress":      "support@webscreenshot.ifempty.de",
		"countrycode":       "de_DE",
		"currencycode":      "EUR",
		"currencysymbol":    "€",
		"toScreenshot":      "Zum Screenshot",
		"price":             "5.99",
	})
}

func indexEN(g *gin.Context) {
	g.HTML(200, "index.tmpl", gin.H{
		"title":             "Screenshot Website",
		"needmoresnaphosts": "Need more than 10 snapshots ?",
		"gopro1":            "Go Professional for",
		"gopro2":            "Get 1000 screenshots per month & unlimited image sharing",
		"emailaddress":      "support@webscreenshot.ifempty.de",
		"anyquestions":      "any questions? write us",
		"countrycode":       "en_US",
		"currencycode":      "USD",
		"currencysymbol":    "$",
		"toScreenshot":      "show screenshot",
		"price":             "5.99",
	})
}
