package main

import (
	"bytes"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
)

var storage *s3.S3

type RecordDoneInput struct {
	App     string `form:"app"`
	Address string `form:"addr"`
	Name    string `form:"name"`
	Path    string `form:"path"`
}

func main() {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" || appEnv == "local" {
		err := godotenv.Load(".env")
		if err != nil {
			panic(err)
		}
	}

	config := &aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
		Credentials: credentials.NewStaticCredentials(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"",
		),
		CredentialsChainVerboseErrors: aws.Bool(true),
	}

	sess, err := session.NewSession(config)
	if err != nil {
		panic(err)
	}

	storage = s3.New(sess)

	api := echo.New()
	api.POST("/record_done", recordDone)

	if err := api.Start(":8000"); err != nil {
		panic(err)
	}
}

func recordDone(c echo.Context) error {
	input := RecordDoneInput{}
	if err := (&echo.DefaultBinder{}).BindBody(c, &input); err != nil {
		log.Println("failed to bind body", err)
		return c.JSON(http.StatusBadRequest, err.Error())
	}

	file, err := os.Open(input.Path)
	if err != nil {
		log.Println("failed to open file", err)
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		log.Println("failed to get stat from file", err)
		return err
	}

	size := fileInfo.Size()
	buffer := make([]byte, size)

	_, err = file.Read(buffer)
	if err != nil {
		log.Println("failed to read buffer from file", err)
		return err
	}

	_, err = storage.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(os.Getenv("S3_BUCKET_NAME")),
		Key:                  aws.String(input.Path),
		ACL:                  aws.String("private"),
		Body:                 bytes.NewReader(buffer),
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(http.DetectContentType(buffer)),
		ContentDisposition:   aws.String("attachment"),
		ServerSideEncryption: aws.String("AES256"),
	})

	if err != nil {
		log.Println("failed to upload file", err)
		return err
	}

	if err := os.RemoveAll(input.Path); err != nil {
		log.Println("failed to remove file", err)
		return err
	}

	return nil
}
