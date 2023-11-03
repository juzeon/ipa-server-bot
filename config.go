package main

import (
	"gopkg.in/yaml.v3"
	"os"
)

type S3 struct {
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	Endpoint        string `yaml:"endpoint"`
	Bucket          string `yaml:"bucket"`
	Region          string `yaml:"region"`
	URL             string `yaml:"url"`
}
type Config struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
	S3     S3     `yaml:"s3"`
}

func NewConfig() *Config {
	v, err := os.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}
	var config Config
	err = yaml.Unmarshal(v, &config)
	if err != nil {
		panic(err)
	}
	return &config
}
