package main

import (
	"log"
	"mime"
	"net/smtp"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/jordan-wright/email"
	"github.com/satori/go.uuid"
)

func sendmail(to, subject, text, filename string) {
	e := email.NewEmail()
	e.From = "WebScreenshot <webscreenshot@webscreenshot.ifempty.de>"
	e.To = []string{to}
	e.Text = []byte(text)
	e.Subject = subject
	e.AttachFile(filename)
	e.Send("smtp.mailgun.org:587",
		smtp.PlainAuth("", "postmaster@webscreenshot.ifempty.de",
			appSettings.SMTPPassword, "smtp.mailgun.org"))
}

func bucketinfo(g *gin.Context) {
	b := GetBucketUrl("nightcrawlerlinks")
	g.String(200, b)
}

func getMimeType(fileName string) string {
	ext := path.Ext(fileName)
	if ext == ".jpeg" {
		ext = ".jpg"
	}
	cType := mime.TypeByExtension(ext)
	if cType == "" {
		cType = "binary/octet-stream"
	}
	return cType
}

func GetBucketUrl(bucketName string) string {
	return "https://s3.amazonaws.com/" + bucketName + "/"
}

func uploadToS3(fileName string, key string, meta map[string]*string) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	svc := s3.New(session.New(), &aws.Config{Region: aws.String("us-east-1")})

	bucketName := appSettings.S3Buckets["Screenshots"]
	cType := getMimeType(key)

	_, err = svc.PutObject(&s3.PutObjectInput{
		Body:        f,
		Bucket:      &bucketName,
		Key:         &key,
		ContentType: &cType,
		Metadata:    meta,
	})

	if err != nil {
		log.Println(err)
	}

	bucketUrl := GetBucketUrl(bucketName)
	return bucketUrl + key, nil
}

func siteinfodyn(g *gin.Context) {

}

func IsValidScreenSize(input string, maxSize int) bool {
	splitted := strings.Split(input, "x")
	if len(splitted) != 2 {
		return false
	}

	sizeX, err := strconv.Atoi(splitted[0])
	if err != nil || sizeX <= 0 || sizeX > maxSize {
		return false
	}
	sizeY, err := strconv.Atoi(splitted[1])
	if err != nil || sizeY <= 0 || sizeY > maxSize {
		return false
	}
	return true
}

func queueScreenshot(g *gin.Context) {
	queryUrl := g.Query("url")
	format := g.Query("format")
	screensize := g.Query("screensize")
	email := g.Query("email")

	if format == "" {
		format = "jpg"
	}
	if queryUrl == "" {
		g.String(403, "needs url parameter")
		return
	}
	if screensize == "" || !IsValidScreenSize(screensize, 4000) {
		screensize = "12080x800"
	}

	url, err := url.Parse(queryUrl)
	if err != nil {
		g.String(403, "invalid url")
		return
	}
	if !url.IsAbs() {
		g.String(403, "url not absolute")
		return
	}
	if url.Host == "localhost" || url.Host == "127.0.0.1" || url.Host == ":::1" {
		g.String(403, "cant crawl localhost")
		return
	}

	fileUUID := uuid.NewV4()
	fname := fileUUID.String() + "." + format

	task := TaskElement{
		Id:         fileUUID,
		Func:       doScreenshot,
		Param1:     url.String(),
		Param2:     email,
		Param3:     fname,
		Additional: []interface{}{format, screensize},
	}

	tasks <- &task
	g.JSON(200, "ok")
}

func doScreenshot(t *TaskElement) {
	queryUrl := t.Param1
	email := t.Param2
	fname := t.Param3
	format := t.Additional[0].(string)
	screensize := t.Additional[1].(string)

	_, err := runPhantom("screen-capture.js", queryUrl, fname,
		format, screensize)
	if err != nil {
		log.Println(err)
		t.Message = err.Error()
		t.ErrorCode = 2
		return
	}

	if email != "" {
		sendmail(email, "sreenshot "+queryUrl, "screenshot from url "+queryUrl,
			"./"+fname)
		t.Success = true
	} else {
		meta := map[string]*string{
			"URL": &queryUrl,
		}
		downloadUrl, _ := uploadToS3("./"+fname, fname, meta)
		t.Success = true
		t.Message = downloadUrl
	}
	os.Remove("./" + fname)
}

func runQueue() {
	for task := range tasks {
		task.Func(task)
	}
}

func runPhantom(args ...string) ([]byte, error) {
	out, err := exec.Command(appSettings.PhantomPath+"phantomjs", args...).Output()
	return out, err
}

func crawltask(g *gin.Context) {

}
