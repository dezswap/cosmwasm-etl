package s3client

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/assert"
)

const chainId string = "phoenix-1"

var testRawTxByte []byte
var client *s3ClientInfo

func makeTxByte() {
	var err error
	testRawTxByte, err = base64.StdEncoding.DecodeString("CqgCCqUCCiQvY29zbXdhc20ud2FzbS52MS5Nc2dFeGVjdXRlQ29udHJhY3QS/AEKLHRlcnJhMTR2ajZlZDRoZ203ZHY5NGR6NzY5NjRnM2x4bDV3ajk1amFmcGw4EkB0ZXJyYTF6NzcwNXQycDVwNnJlbDkzZmQ3enJzaDhhNGx1eHl4ejg4YTR6a21sY3R3ZjM4eWg1MjBxdDk0OW44GnZ7InN3YXAiOnsibWluaW11bV9yZWNlaXZlIjoiOTg1MTQyMTkiLCJvZmZlcl9hc3NldCI6eyJhbW91bnQiOiIxMDAwMDAwMDAiLCJpbmZvIjp7Im5hdGl2ZV90b2tlbiI6eyJkZW5vbSI6InVsdW5hIn19fX19KhIKBXVsdW5hEgkxMDAwMDAwMDASaApQCkYKHy9jb3Ntb3MuY3J5cHRvLnNlY3AyNTZrMS5QdWJLZXkSIwohA4RvoQ6AJPcezRNaBc8IK6qS0iTJUsikM6AuVJfGLCfdEgQKAggBGBcSFAoOCgV1bHVuYRIFOTAxNDUQg9ckGkD7+6yMOW70wuuXu1tIcMBBOGHYnY1mBbeBI5AJcK3k6CfobItfYil9pJcxpkvZ33Jxlk3r6xISEJDpfGTi0Evc")
	if err != nil {
		panic(err)
	}
}

func (client *s3ClientInfo) deleteFromS3(path ...string) error {
	_, err := client.s3Client.DeleteObject(
		context.TODO(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(client.bucket),
			Key:    aws.String(strings.Join(path, "/")),
		},
	)

	return err
}

func TestUploadToS3AndGet(t *testing.T) {
	// setup
	var err error
	makeTxByte()

	client, err = NewMockClient()
	if err != nil {
		panic(err)
	}
	// end of setup

	// start upload test
	filename := "testfile.json"

	err = client.UploadFileToS3(testRawTxByte, chainId, filename)
	assert.NoError(t, err)

	s3obj, err := client.GetFileFromS3(chainId, filename)
	assert.Equal(t, testRawTxByte, s3obj)
	assert.NoError(t, err)
	// end of the test

	// teardown
	err = client.deleteFromS3(chainId, filename)
	if err != nil {
		panic(err)
	}
}

func TestChangeLatestBlock(t *testing.T) {
	// setup
	var err error
	var currBlock int64

	makeTxByte()

	client, err = NewMockClient()
	if err != nil {
		panic(err)
	}
	// end of setup

	// 1. No block stored. 0 should be returned
	{
		currBlock, err = client.GetLatestProcessedBlockNumber(chainId)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), currBlock)
	}

	// 2. Upload the block 1 and check the latest block getter
	{
		err = client.ChangeLatestBlock(currBlock+1, chainId)
		assert.NoError(t, err)

		currBlock, err = client.GetLatestProcessedBlockNumber(chainId)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), currBlock)
	}

	// 3. Upload the block 2, unmark the latest to block 1, and check the latest block getter
	{
		err = client.ChangeLatestBlock(currBlock+1, chainId)
		assert.NoError(t, err)

		currBlock, err = client.GetLatestProcessedBlockNumber(chainId)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), currBlock)
	}

	// teardown
	tearDown()
	// end of teardown
}

func tearDown() {
	client.MakePagenator()

	filelist, err := client.GetWholeBlockFileListFromS3(chainId)
	if err != nil {
		panic(err)
	}

	for _, unitfile := range filelist {
		err = client.deleteFromS3(unitfile)

		if err != nil {
			panic(err)
		}
	}
}
