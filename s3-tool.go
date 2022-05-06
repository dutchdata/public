package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	s3ssion *s3.S3
	cbuffer int
	cFile []string
	sFile string
)

// type ObjectRecord struct {
// 	Key string `json:"key"`
// 	KSize int `json:"k_size"`
// 	StorageClass string `json:"storage_class"`
// 	LastkModified string `json:"last_modified"`
// }

type BucketRecord struct {
	Name string `json:"name"`
	ObjectCount int `json:"object_count"`
	TotalSize int64 `json:"total_size_k"`
}

func main() {

	e := echo.New()
	e.GET("/",func(c echo.Context) (error) {
		return c.String(http.StatusOK, sFile)
	})

	e.GET("/auth", accessKeyHandler)
	e.GET("/go", sessionHandler)
	e.GET("/get", recordHandler)

	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Logger.Fatal(e.Start(":8080"))
}

func startSession() (s3ssion *s3.S3) {
	s3ssion = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})))
	return s3ssion
}

func recordHandler(c echo.Context) error {
	sFile := getBucketRecords()
	return c.String(http.StatusOK,sFile)
}

func sessionHandler(c echo.Context) error {

	return c.String(http.StatusOK,"go")
}

func accessKeyHandler(c echo.Context) (error) {

	type keySet struct {
		Access_key_id string `json:"access_key_id"`
		Secret_key string `json:"secret_key"`
	}

	key_id := c.QueryParam("access_key_id")
	key_id_string := url.QueryEscape(key_id)
	os.Setenv("AWS_ACCESS_KEY_ID",key_id_string)

	secret_key := c.QueryParam("secret_key") 
	secret_key_string := url.QueryEscape(secret_key)
	os.Setenv("AWS_SECRET_ACCESS_KEY",secret_key_string)

	keys := keySet{
		Access_key_id: key_id_string,
		Secret_key: secret_key_string,
	}

	keys_b, _ := json.Marshal(keys)
	key_set := string(keys_b)

	fmt.Println(key_set)
	
	return c.String(http.StatusOK, key_set)
}

func listBuckets() (bucket_count int, bucket_list []string) {
	resp, err := s3ssion.ListBuckets(&s3.ListBucketsInput{})

	if err != nil {
		panic(err)
	}

	bucket_count = len(resp.Buckets)
	bucket_list = make([]string,bucket_count)
	
	for i := 0; i < bucket_count; i++ {
		bucket_name := *resp.Buckets[i].Name
		bucket_list[i] = bucket_name
	}

	return bucket_count, bucket_list
}

func listObjects(bucket string, c chan BucketRecord) () {
	resp, err := s3ssion.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		panic(err)
	}

	// get collective size of objects for bucket in k
	contents := resp.Contents
	var bucket_size int64
	for i := range contents {
		bucket_size += *contents[i].Size
	}

	// create BucketRecord object to send via channel
	bucket_record := BucketRecord{
		Name: *resp.Name,
		ObjectCount: len(resp.Contents),
		TotalSize: bucket_size,
	} 
	// send BucketRecord object back to caller via channel
	c <- bucket_record
	if cbuffer == 0 {
		close(c)
	}
}

func getBucketRecords() (sFile string) {
	// start timer for operation(s)
	start := time.Now()
	// start session (concurrent read ok; 
	// concurrent write not recommended)
	s3ssion = startSession()
	// run listBuckets()
	bucket_count, bucket_list := listBuckets()
	// define buffer and goroutine channel range
	cbuffer = bucket_count

	// make a channel to receive output from listObjects() calls
	ch := make(chan BucketRecord,bucket_count)

	// start a goroutine call to listObjects() for each bucket 
	// returned by listBuckets()
	for i := range bucket_list {
		go listObjects(bucket_list[i],ch)
	}

	// empty cFile and eFile for idempotency with repeated calls
	cFile = []string{}
	sFile = ""

	// start receiver loop for channel of BucketRecord
	for i := range ch {
		b, _ := json.Marshal(i)
		s := string(b)
		fmt.Println(s)
		cFile = append(cFile,s)
		// fmt.Println(time.Since(start),i)

		// decrement cbuffer and break loop when cbuffer is 0
		cbuffer --
		if cbuffer == 0 {
			break
		}
	}	
	fmt.Println("total:",time.Since(start))
	b, _ := json.Marshal(cFile)
	sFile = string(b)
	return sFile
}