package configs

import "github.com/spf13/viper"

type S3Config struct {
	Bucket string
	Region string
	Key    string
	Secret string
}

func s3Config(v *viper.Viper) S3Config {
	return S3Config{
		Bucket: v.GetString("s3.bucket"),
		Region: v.GetString("s3.region"),
		Key:    v.GetString("s3.key"),
		Secret: v.GetString("s3.secret"),
	}
}
