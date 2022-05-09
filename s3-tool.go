package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
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
	keys KeySet
	rows [][]string
	target_file_name string
	target_file_directory string
)

// type ObjectRecord struct {
// 	Key string `json:"key"`
// 	KSize int `json:"k_size"`
// 	StorageClass string `json:"storage_class"`
// 	LastkModified string `json:"last_modified"`
// }

type KeySet struct {
	Access_key_id string `json:"access_key_id"`
	Secret_key string `json:"secret_key"`
	Region string `json:"region"`
}

type BucketRecord struct {
	Name string `json:"name"`
	ObjectCount int `json:"object_count"`
	TotalSize int64 `json:"total_size_k"`
}

func main() {

	e := echo.New()
	e.GET("/",func(c echo.Context) (error) {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/auth", accessKeyHandler)
	e.GET("/go", recordHandler)
	e.GET("/get",downloadHandler)

	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Logger.Fatal(e.Start(":8080"))
}

func startSession() (s3ssion *s3.S3) {

	keys_region := os.Getenv("AWS_DEFAULT_REGION")
	if keys_region == "" {
		keys_region = "us-west-2"
	}
	s3ssion = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String(keys_region),
	})))
	return s3ssion
}

func downloadHandler(c echo.Context) error {
	target_file_name = "output.csv"
	target_file_directory = "s3-tool-output"
	writeRecords(rows,target_file_name,target_file_directory)
	return c.Attachment(target_file_name,target_file_name)
}

func recordHandler(c echo.Context) error {
	rows = getBucketRecords()
	b, _ := json.Marshal(rows)
	s := string(b)
	return c.String(http.StatusOK,s)
}

func accessKeyHandler(c echo.Context) (error) {

	key_id := c.QueryParam("access_key_id")
	key_id_string := url.QueryEscape(key_id)
	os.Setenv("AWS_ACCESS_KEY_ID",key_id_string)

	secret_key := c.QueryParam("secret_key") 
	secret_key_string := url.QueryEscape(secret_key)
	os.Setenv("AWS_SECRET_ACCESS_KEY",secret_key_string)

	region := c.QueryParam("region") 
	region_string := url.QueryEscape(region)
	os.Setenv("AWS_DEFAULT_REGION",region_string)

	keys = KeySet{
		Access_key_id: key_id_string,
		Secret_key: secret_key_string,
		Region: region_string,
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

func getBucketRecords() (rows [][]string) {
	start := time.Now() // start timer for operation(s)
	s3ssion = startSession() // start session 
	bucket_count, bucket_list := listBuckets() // get bucket names
	cbuffer = bucket_count // define buffer range

	// make a channel to receive BucketRecord objects
	ch := make(chan BucketRecord,bucket_count) 

	// start a goroutine for each bucket available
	for i := range bucket_list {
		go listObjects(bucket_list[i],ch)
	}
	// receive records from channel and write to output file
	for i := range ch {
		row := recordSerializer(i)
		rows = append(rows,row)
		cbuffer -- // decrement cbuffer and break loop when == 0
		if cbuffer == 0 {
			break
			return rows
		}
	}
	// writeRecords(rows,"test-data01010101.csv","s3-tool-output")
	fmt.Println("total:",time.Since(start)) // log total request time
	return rows
}

func recordSerializer(record BucketRecord) (row []string) {
	rowName := record.Name
	rowObjectCount := strconv.Itoa(record.ObjectCount)
	rowTotalSize := strconv.Itoa(int(record.TotalSize))
	row = []string{rowName,rowObjectCount,rowTotalSize}
	return row
}

func writeRecords(rows [][]string,file string, directory string) (output_file *os.File) {
	output_file, _ = pathResolver(file,directory)
	defer output_file.Close()
	writer := newRecordWriter(output_file,[]string{"name","object_count","total_size_k"})
	for i := range rows {
		writer.Write(rows[i])
	}
	writer.Flush()
	return output_file
}

func newRecordWriter(file *os.File,headers []string) (writer *csv.Writer) {
	writer = csv.NewWriter(file)
	writer.Write(headers)
	return writer
}

func pathResolver(target_file_name string, parent_directory string) (file *os.File, path string) {
	root_directory, _ := os.UserHomeDir()
	os.Mkdir(root_directory + "/" + parent_directory,0755)
	path = root_directory + "/" + parent_directory + "/" + target_file_name
	file, _ = os.Create(path)
	return file, path
}