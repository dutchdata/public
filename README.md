# s3-tool 
###### v0.1.0

### Getting Started (console & browser)

1. `docker pull dutch10/s3-tool:0.1.0`
2. `docker run -p 8080:8080 dutch10/s3-tool:0.1.0`
3. Server will start and listen on port 8080 (TCP).
4. Open browser to `http://localhost:8080/`

#### Authenticating

5. `http://localhost:8080/auth` is the endpoint.
6. Use query parameters `access_key_id`, `secret_key`, and `region` to pass your Access Key ID, Secret Key ID, and default AWS region, respectively and in the same request. This sets ENV variables in the container for AWS authentication (the variables are lost when the container stops running).

#### Running the S3 Analysis

7. `http://localhost:8080/go` will start the analysis, which (for now) includes only the number of buckets, the number of objects per bucket, and the collective size of those objects. It starts one goroutine per bucket, logs the result of each as it completes, and then displays all the results at once in the webpage (it's all function and no form, right now). 
8. `http://localhost:8080/getcsv` will prompt you to download the CSV file with the bucket data retrieved.
9. `http://localhost:8080/getrec` will prompt you to download the Recommendation file. Currently, the only recommendation given will be to either turn on CloudTrail events, or to wait until the next major version of this applet. 

## Next Major Version

In the next major version, the applet will parse CloudTrail data events and display the number of daily GetObject, PutObject, and DeleteObject events for the lifetime of each bucket. 
