package configs

type S3Config struct {
	Bucket string `mapstructure:"bucket"`
	Region string `mapstructure:"region"`
	Key    string `mapstructure:"key"`
	Secret string `mapstructure:"secret"`
}
