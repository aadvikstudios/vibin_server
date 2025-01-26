package services

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		panic(err)
	}
	s3Client = s3.NewFromConfig(cfg)
}

// GenerateUploadURL generates a presigned URL for uploading a file
func GenerateUploadURL(fileName, fileType string) (string, string, error) {
	key := "profile-pics/" + time.Now().Format("20060102150405") + "-" + fileName
	params := &s3.PutObjectInput{
		Bucket:      aws.String(os.Getenv("S3_BUCKET_NAME")),
		Key:         aws.String(key),
		ContentType: aws.String(fileType),
	}
	presigner := s3.NewPresignClient(s3Client)
	presignedURL, err := presigner.PresignPutObject(context.TODO(), params, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return "", "", err
	}
	return presignedURL.URL, key, nil
}

// GenerateReadURL generates a presigned URL for reading a file
func GenerateReadURL(key string) (string, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("S3_BUCKET_NAME")),
		Key:    aws.String(key),
	}
	presigner := s3.NewPresignClient(s3Client)
	presignedURL, err := presigner.PresignGetObject(context.TODO(), params, s3.WithPresignExpires(5*time.Minute))
	if err != nil {
		return "", err
	}
	return presignedURL.URL, nil
}
