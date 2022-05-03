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
	// Objects ObjectRecord `json:"objects"`
}

func main() {
	start := time.Now()
	s3ssion = startSession()
	bucket_count, bucket_list := listBuckets()
	cbuffer = bucket_count
	// make a channel to receive output from ListObjects() calls
	c := make(chan *s3.ListObjectsV2Output,bucket_count)
	// make a channel to receive output from getBucketSize() calls
	// ch := make(chan int,bucket_count)
	// start a goroutine call to listObjects() for each bucket returned by listBuckets
	for i := range bucket_list {
		go listObjects(bucket_list[i],c)
		// go getBucketSize(bucket_list[i],c)
	}
	// start receiver loop 
	for i := range c {
		bucket_record := BucketRecord{
			Name: *i.Name, 
			ObjectCount: len(i.Contents),
			// BucketSize: getBucketSize()
		}
		b, _ := json.Marshal(bucket_record)
		s := string(b)
		fmt.Println(s,time.Since(start),i)

		getBucketSize(*i.Name)
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

func listObjects(bucket string, c chan *s3.ListObjectsV2Output) () {
	resp, err := s3ssion.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		panic(err)
	}
	c <- resp
	if cbuffer == 0 {
		close(c)
	}
}

func getBucketSize(bucket string) (bucket_size int64) {
	resp, err := s3ssion.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		panic(err)
	}
	contents := resp.Contents
	for i := range contents {
		bucket_size += *contents[i].Size
	}
	fmt.Println(bucket_size)
	return bucket_size
}
