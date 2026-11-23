package main

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
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

var configPath = "config.yml"

func NewConfig() *Config {
	tightenOwnerOnly(configPath, 0o600)
	v, err := os.ReadFile(configPath)
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

func tightenOwnerOnly(path string, mode os.FileMode) {
	if err := os.Chmod(path, mode); err != nil {
		slog.Error("cannot restrict file permissions", "path", path, "err", err)
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		slog.Error("cannot stat file permissions", "path", path, "err", err)
		return
	}
	if info.Mode().Perm()&0o077 != 0 {
		slog.Error("file remains readable by group or others", "path", path, "mode", info.Mode().Perm())
	}
}
