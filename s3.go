package main

import (
	"bytes"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"strings"
	"time"
)

var minioClient *minio.Client

func SetupS3() {
	client, err := minio.New(config.S3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.S3.AccessKeyID, config.S3.SecretAccessKey, ""),
		Secure: true,
		Region: config.S3.Region,
	})
	if err != nil {
		panic(err)
	}
	minioClient = client
}

func UploadS3(v []byte, filename string, contentType string) (string, error) {
	ctx, cancel := CreateTimeoutContext(120 * time.Second)
	defer cancel()
	info, err := minioClient.PutObject(ctx, config.S3.Bucket, filename, bytes.NewReader(v), int64(len(v)),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	result := config.S3.URL
	if !strings.HasSuffix(result, "/") {
		result += "/"
	}
	result += info.Key
	return result, nil
}
