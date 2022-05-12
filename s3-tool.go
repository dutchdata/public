package main

import (
	"bytes"
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
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/s3"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	s3ssion *s3.S3
	Cbuffer int
	Keys KeySet
	Rows [][]string
	Target_file_name string
	Target_file_directory string
	Output_file *os.File
	Path string
)

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

type Trail struct {
	Name string `json:"name"`
	Bucket string `json:"bucket"`
}

type CloudTrailCustomEvent struct {
	UserIdentity CloudTrailUserIdentity `json:"userIdentity,omitempty"`
	EventType string `json:"eventType,omitempty"`
	EventId string `json:"eventID,omitempty"`
	AddEventData AddEventData `json:"additionalEventData"`
	EventTime time.Time `json:"eventTime,omitempty"`
	EventSource string `json:"eventSource,omitempty"`
	EventName string `json:"eventName,omitempty"`
	EventRegion string `json:"awsRegion,omitempty"`
	SourceIP string `json:"sourceIPAddress,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	RequestParameters RequestParameters `json:"requestParameters,omitempty"`
}

type CloudTrailUserIdentity struct {
	Type string `json:"type,omitempty"`
	InvokedBy string `json:"invokedBy,omitempty"`
}

type RequestParameters struct {
	BucketName string `json:"bucketName,omitempty"`
	Host string `json:"Host,omitempty"`
	Acl string `json:"acl,omitempty"`
}

type AddEventData struct {
	BytesIn int `json:"bytesTransferredIn"`
	BytesOut int `json:"bytesTransferredOut"`
}

func main() {

	e := echo.New()
	e.GET("/",func(c echo.Context) (error) {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/auth", accessKeyHandler)
	e.GET("/go", recordHandler)
	e.GET("/get", downloadHandler)
	e.GET("/check-for-trails",trailCheckHandler)
	e.GET("/get-trail-events", trailEventHandler)

	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Logger.Fatal(e.Start(":8080"))
}

func checkForTrails()(trails []Trail, rows [][]string){
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	
	// Create CloudTrail client
	svc := cloudtrail.New(sess)	

	// call DescribeTrails()
	resp, err := svc.DescribeTrails(&cloudtrail.DescribeTrailsInput{TrailNameList: nil})
	if err != nil {
    	fmt.Println("Got error calling CreateTrail:")
    	fmt.Println(err.Error())
    	return
	}
	
	// list results of search for trails 
	fmt.Println("Found", len(resp.TrailList), "trail(s)")
	
	// list data about trails, if exist
	trails = []Trail{}
	rows = [][]string{}
	for _, trail := range resp.TrailList {

		t := Trail{Name: *trail.Name,Bucket:*trail.S3BucketName}
		trails = append(trails,t)
		
		row := []string{*trail.Name,*trail.S3BucketName}
		rows = append(rows,row)
	}

	fmt.Println(rows)
	return trails, rows
}

func checkForEvents()() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := cloudtrail.New(sess)

	input := &cloudtrail.LookupEventsInput{EndTime: aws.Time(time.Now())}
	
	resp, err := svc.LookupEvents(input)
	if err != nil {
		fmt.Println("Got error calling CreateTrail:")
		fmt.Println(err.Error())
		return
	}

	fmt.Println("Found", len(resp.Events), "events before now")
	fmt.Println("")

	for _, event := range resp.Events {
		m := []byte(*event.CloudTrailEvent)
		r := bytes.NewReader(m)
		decoder := json.NewDecoder(r)
		ce := &CloudTrailCustomEvent{}
		err := decoder.Decode(ce)
		if err != nil {
			panic(err)
		}
		// fmt.Println(*ce)
	}

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

func trailCheckHandler(c echo.Context) error {
	checkForTrails()
	return c.String(http.StatusOK,"check log")
}

func trailEventHandler(c echo.Context) error {
	checkForEvents()
	return c.String(http.StatusOK,"check log")
}

func downloadHandler(c echo.Context) error {
	Target_file_name = "output.csv"
	Target_file_directory = "s3-tool-output"
	headers := []string{"name","object_count","total_size_k"}
	writeRecords(headers,Rows,Target_file_name,Target_file_directory)
	fmt.Println("File downloaded to",Path)
	return c.Attachment(Path,"output.csv")
}

func recordHandler(c echo.Context) error {
	Rows = getBucketRecords()
	b, _ := json.Marshal(Rows)
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

	Keys = KeySet{
		Access_key_id: key_id_string,
		Secret_key: secret_key_string,
		Region: region_string,
	}

	keys_b, _ := json.Marshal(Keys)
	key_set := string(keys_b)
	
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
	// if Cbuffer == 0 {
	// 	close(c)
	// }
}

func getBucketRecords() (Rows [][]string) {
	start := time.Now() // start timer for operation(s)
	s3ssion = startSession() // start session 
	bucket_count, bucket_list := listBuckets() // get bucket names
	Cbuffer = bucket_count // define buffer range

	// make a channel to receive BucketRecord objects
	ch := make(chan BucketRecord,bucket_count) 

	// start a goroutine for each bucket available
	for i := range bucket_list {
		go listObjects(bucket_list[i],ch)
	}
	// receive records from channel and write to output file
	for i := range ch {
		row := recordSerializer(i)
		Rows = append(Rows,row)
		Cbuffer -- // decrement cbuffer and break loop when == 0
		if Cbuffer == 0 {
			break
			return Rows
		}
	}
	fmt.Println("getBucketRecords() call time:",time.Since(start)) // log total request time
	return Rows
}

func recordSerializer(record BucketRecord) (row []string) {
	rowName := record.Name
	rowObjectCount := strconv.Itoa(record.ObjectCount)
	rowTotalSize := strconv.Itoa(int(record.TotalSize))
	row = []string{rowName,rowObjectCount,rowTotalSize}
	return row
}

func writeRecords(headers []string,rows [][]string,file string, directory string) (output_file *os.File) {
	output_file, Path = pathResolver(file,directory)
	defer output_file.Close()
	writer := newRecordWriter(output_file,headers)
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

func pathResolver(target_file_name string, parent_directory string) (file *os.File, Path string) {
	root_directory, _ := os.UserHomeDir()
	os.Mkdir(root_directory + "/" + parent_directory,0755)
	Path = root_directory + "/" + parent_directory + "/" + target_file_name
	file, _ = os.Create(Path)
	return file, Path
}