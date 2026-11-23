package main

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

func validObjectKey(key string) bool {
	if key == "" || len(key) > 1024 || strings.Contains(key, "\\") || strings.HasPrefix(key, "/") {
		return false
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func UploadS3(r io.Reader, size int64, filename string, contentType string) (string, error) {
	if !validObjectKey(filename) || strings.ContainsAny(contentType, "\r\n") || size < 0 {
		return "", errors.New("invalid upload")
	}
	ctx, cancel := CreateTimeoutContext(120 * time.Second)
	defer cancel()
	_, err := minioClient.PutObject(ctx, config.S3.Bucket, filename, r, size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", err
	}
	result := config.S3.URL
	if !strings.HasSuffix(result, "/") {
		result += "/"
	}
	result += filename
	return result, nil
}
