package s3client

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3sdkconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/dezswap/cosmwasm-etl/configs"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
)

const latestTag string = "latest"
const blockTag string = "block"
const latestFilename string = latestTag + ".txt"

var s3Config configs.S3Config

// Interface for actual S3 usage within this module
type s3Interfaces interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// Interface for actual S3 ListObject paginator
type s3ListObjectsV2Pager interface {
	HasMorePages() bool
	NextPage(context.Context, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

var _ s3Interfaces = &s3.Client{}
var _ s3ListObjectsV2Pager = &s3.ListObjectsV2Paginator{}

type S3ClientInterface interface {
	GetLatestProcessedBlockNumber(...string) (int64, error)
	ChangeLatestBlock(int64, ...string) error
	UploadBlockBinary(int64, []byte, ...string) error
	GetBlockFilePath(int64, ...string) []string
	GetFileFromS3(...string) ([]byte, error)
	UploadFileToS3([]byte, ...string) error
}

// actual service client contructor
type s3ClientInfo struct {
	s3Client s3Interfaces
	bucket   string

	// Just uses for making interface of s3 Get object paginator
	paginator s3ListObjectsV2Pager
}

var _ S3ClientInterface = &s3ClientInfo{}

func NewClient() (S3ClientInterface, error) {
	s3Config = configs.Get().S3

	cred := credentials.NewStaticCredentialsProvider(
		s3Config.Key,    // user,
		s3Config.Secret, // key,
		"",              // let it be empty
	)

	cfg, err := s3sdkconfig.LoadDefaultConfig(
		context.TODO(),
		s3sdkconfig.WithCredentialsProvider(cred),
		s3sdkconfig.WithRegion(s3Config.Region),
	)

	if err != nil {
		err = errors.Wrap(err, "S3 client create")
		return nil, err
	}

	ret := &s3ClientInfo{
		s3Client: s3.NewFromConfig(cfg),
		bucket:   s3Config.Bucket,
	}

	return ret, nil
}

func (client *s3ClientInfo) GetClient() s3Interfaces {
	return client.s3Client
}

func (client *s3ClientInfo) GetBucket() string {
	return client.bucket
}

// Upload the given binary into the given path
func (client *s3ClientInfo) UploadFileToS3(data []byte, path ...string) error {
	_, err := client.GetClient().PutObject(
		context.Background(),
		&s3.PutObjectInput{
			Bucket: aws.String(client.GetBucket()),
			Key:    aws.String(strings.Join(path, "/")),
			ACL:    s3types.ObjectCannedACLBucketOwnerFullControl,
			Body:   bytes.NewReader(data),
		},
	)

	if err != nil {
		err = errors.Wrap(err, "UploadFileToS3")
	}

	return err
}

// Get the binary from the given path
func (client *s3ClientInfo) GetFileFromS3(path ...string) ([]byte, error) {
	resp, err := client.GetClient().GetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(client.GetBucket()),
			Key:    aws.String(strings.Join(path, "/")),
		},
	)

	if err != nil {
		err = errors.Wrap(err, "GetFileFromS3, S3 object getter")
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		err = errors.Wrap(err, "GetFileFromS3, read buffer")
		return nil, err
	}

	content := buf.String()

	return []byte(content), nil
}

// Before execute GetWholeBlockFileListFromS3, this method should be executed
// FIXME: is there any idea to mock s3.NewListObjectsV2Paginator without mocking whole s3?
func (client *s3ClientInfo) MakePagenator(path ...string) {
	client.paginator = s3.NewListObjectsV2Paginator(
		client.s3Client,
		&s3.ListObjectsV2Input{
			Bucket: aws.String(client.GetBucket()),
			Prefix: aws.String(strings.Join(path, "/")),
		},
	)
}

// Get whole filelist from the given path, as filename
func (client *s3ClientInfo) GetWholeBlockFileListFromS3(path ...string) ([]string, error) {
	ctx := context.TODO()
	filelist := []string{}

	for client.paginator.HasMorePages() {
		resp, err := client.paginator.NextPage(ctx)
		if err != nil {
			err = errors.Wrap(err, "GetWholeBlockFileListFromS3, NextPage")
			return nil, err
		}

		filenameMapped := funk.Map(resp.Contents, func(item s3types.Object) string {
			return *item.Key
		}).([]string)

		filteredFile := funk.Filter(filenameMapped, func(item string) bool {
			return strings.Contains(item, blockTag)
		}).([]string)

		filelist = append(filelist, filteredFile...)
	}

	return filelist, nil
}

func (client *s3ClientInfo) GetBlockFilePath(blockNum int64, folderPath ...string) []string {
	filename := fmt.Sprintf("%s_%d.json", blockTag, blockNum)
	return append(folderPath, filename)
}

func (client *s3ClientInfo) UploadBlockBinary(blockNum int64, data []byte, folderPath ...string) error {
	pathFile := client.GetBlockFilePath(blockNum, folderPath...)

	// Upload just blank file
	err := client.UploadFileToS3(data, pathFile...)

	return err
}

// if return is 0, it means no block collected yet
// if return is -1, there is an error
// try to get & open latest.txt file and open the number
func (client *s3ClientInfo) GetLatestProcessedBlockNumber(folderPath ...string) (int64, error) {
	path := append(folderPath, latestFilename)

	resp, err := client.GetFileFromS3(path...)
	if err != nil && strings.Contains(err.Error(), "not found") {
		return 0, nil
	} else if err != nil {
		err = errors.Wrap(err, "GetLatestProcessedBlockNumber, GetFileFromS3")
		return -1, err
	}

	respStr := strings.TrimSpace(string(resp))
	blockNo, err := strconv.ParseInt(respStr, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "GetLatestProcessedBlockNumber, ParseInt")
		return -1, err
	}

	return blockNo, nil
}

// No rename feature in S3. Need copy with rename & delete the old one
func (client *s3ClientInfo) ChangeLatestBlock(currBlockNum int64, folderPath ...string) error {
	var err error
	nextFilePath := strings.Join(append(folderPath, latestFilename), "/")

	_, err = client.s3Client.PutObject(
		context.TODO(),
		&s3.PutObjectInput{
			Bucket: aws.String(client.GetBucket()),
			Key:    aws.String(nextFilePath),
			ACL:    s3types.ObjectCannedACLBucketOwnerFullControl,
			Body:   bytes.NewReader([]byte(strconv.FormatInt(currBlockNum, 10))),
		},
	)

	if err != nil {
		err = errors.Wrap(err, "MarkLatestBlock, UploadFileToS3")
	}

	return err
}
