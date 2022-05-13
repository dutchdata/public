package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudtrail"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	S3Session *s3.S3
	Cbuffer int
)

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

func CheckForTrails()(trails []Trail, rows [][]string){
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

func CheckForEvents()() {
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

func startSession() (S3Session *s3.S3) {

	keys_region := os.Getenv("AWS_DEFAULT_REGION")
	if keys_region == "" {
		keys_region = "us-west-2"
	}
	S3Session = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String(keys_region),
	})))
	return S3Session
}

func listBuckets() (bucket_count int, bucket_list []string) {
	resp, err := S3Session.ListBuckets(&s3.ListBucketsInput{})

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
	resp, err := S3Session.ListObjectsV2(&s3.ListObjectsV2Input{
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

func recordSerializer(record BucketRecord) (row []string) {
	rowName := record.Name
	rowObjectCount := strconv.Itoa(record.ObjectCount)
	rowTotalSize := strconv.Itoa(int(record.TotalSize))
	row = []string{rowName,rowObjectCount,rowTotalSize}
	return row
}

func GetBucketRecords() (Rows [][]string) {
	start := time.Now() // start timer for operation(s)
	S3Session = startSession() // start session 
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