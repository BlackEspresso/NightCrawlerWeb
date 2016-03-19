// main.go
package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.Static("/static", "./static")
	r.GET("/screenshot", screenshot)
	r.GET("/", index)
	r.Run(":8080")
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
	g.HTML(200, "index.tmpl", gin.H{})
}
