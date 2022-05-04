package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	s3ssion *s3.S3
	cbuffer int
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
	start := time.Now()
	s3ssion = startSession()
	bucket_count, bucket_list := listBuckets()
	cbuffer = bucket_count

	// make a channel to receive output from ListObjects() calls
	c := make(chan BucketRecord,bucket_count)

	// start a goroutine call to listObjects() for each bucket returned by listBuckets
	for i := range bucket_list {
		go listObjects(bucket_list[i],c)
	}

	// start receiver loop 
	for i := range c {
		b, _ := json.Marshal(i)
		s := string(b)
		fmt.Println(s)
		// fmt.Println(time.Since(start),i)

		// decrement cbuffer and break loop when cbuffer is 0
		cbuffer --
		if cbuffer == 0 {
			break
		}
	}	
	fmt.Println("total:",time.Since(start))
}

func startSession() (s3ssion *s3.S3) {
	s3ssion = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	})))
	return s3ssion
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

