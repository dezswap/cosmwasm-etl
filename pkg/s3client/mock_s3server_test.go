package s3client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Implemented level & limitation within mock service:
//  0. Bucket is not considered

//  1. PutObject - Used Mutex-supporting map and stores the path - binary into there. The bucket is ignored.
//  2. GetObject - Load the binary from the given path. The bucket is ignored.
//  3. ListObjectsV2 - Get filelist from the Mutex map. The restriction - multiple page + 1000 objects per a call - are not implemented yet
//  4. CopyObject - copy the binary from the given source path to the given renamed path
//  5. DeleteObject - delete the key-binary from the mutex map
type mockS3Service struct {
	member sync.Map
}

type mockListObjectsV2Pager struct {
	PageNum int
	Pages   []*s3.ListObjectsV2Output
}

var _ s3Interfaces = &mockS3Service{}
var _ s3ListObjectsV2Pager = &mockListObjectsV2Pager{}

func NewMockClient() (*s3ClientInfo, error) {
	return &s3ClientInfo{
		s3Client:  &mockS3Service{},
		bucket:    "terraswap-test-data",
		paginator: &mockListObjectsV2Pager{},
	}, nil
}

func (m *mockS3Service) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	// error mocking
	if params.Bucket == nil {
		return nil, errors.New("bucket is essential parameter")
	}

	if params.Key == nil {
		return nil, errors.New("key is essential parameter")
	}

	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(params.Body)
	if err != nil {
		return nil, err
	}

	data := buf.Bytes()

	m.member.Store(*params.Key, data)

	return &s3.PutObjectOutput{}, nil
}

func (m *mockS3Service) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if params.Bucket == nil {
		return nil, errors.New("bucket is essential parameter")
	}

	if params.Key == nil {
		return nil, errors.New("key is essential parameter")
	}

	unit, isLoded := m.member.Load(*params.Key)
	if isLoded {
		return &s3.GetObjectOutput{
			Body: io.NopCloser(bytes.NewReader(unit.([]byte))),
		}, nil
	} else {
		return nil, errors.New("not found")
	}
}

func (m *mockS3Service) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if params.Bucket == nil {
		return nil, errors.New("bucket is essential parameter")
	}

	if params.Prefix == nil {
		return nil, errors.New("key is essential parameter")
	}

	contents := []s3types.Object{}
	m.member.Range(func(k, b interface{}) bool {
		if strings.Contains(k.(string), *params.Prefix) {
			unit := s3types.Object{
				Key: aws.String(k.(string)),
			}

			contents = append(contents, unit)
		}

		return true
	})

	return &s3.ListObjectsV2Output{
		Contents: contents,
	}, nil
}

func (m *mockS3Service) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	if params.Bucket == nil {
		return nil, errors.New("bucket is essential parameter")
	}

	if params.CopySource == nil {
		return nil, errors.New("CopySource is essential parameter")
	}

	if params.Key == nil {
		return nil, errors.New("key is essential parameter")
	}

	withoutBuckCopySource := strings.Join(strings.Split(*params.CopySource, "/")[1:], "/")

	unit, isLoded := m.member.Load(withoutBuckCopySource)
	if isLoded {
		m.member.Store(*params.Key, unit)
		return &s3.CopyObjectOutput{}, nil
	} else {
		return nil, errors.New("not found")
	}
}

func (m *mockS3Service) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if params.Bucket == nil {
		return nil, errors.New("bucket is essential parameter")
	}

	if params.Key == nil {
		return nil, errors.New("key is essential parameter")
	}

	_, isExist := m.member.LoadAndDelete(*params.Key)
	if isExist {
		return &s3.DeleteObjectOutput{}, nil
	} else {
		return nil, errors.New("not found")
	}
}

func (l *mockListObjectsV2Pager) HasMorePages() bool {
	return l.PageNum < len(l.Pages)
}

func (l *mockListObjectsV2Pager) NextPage(ctx context.Context, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if l.PageNum >= len(l.Pages) {
		return nil, fmt.Errorf("no more pages")
	}
	output := l.Pages[l.PageNum]
	l.PageNum++
	return output, nil
}
