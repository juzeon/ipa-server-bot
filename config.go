package main

import (
	"errors"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"

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
	Token       string `yaml:"token"`
	APIID       int    `yaml:"api_id"`
	APIHash     string `yaml:"api_hash"`
	SessionFile string `yaml:"session_file"`
	S3          S3     `yaml:"s3"`
}

var configPath = "config.yml"

func parseConfigPath(args []string) (string, error) {
	fs := flag.NewFlagSet("ipa-server-bot", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	path := fs.String("c", "config.yml", "path to the config file")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if *path == "" {
		return "", errors.New("config path is empty")
	}
	return *path, nil
}

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

func (c *Config) validate() error {
	if c == nil || c.Token == "" || c.APIID <= 0 || c.APIHash == "" {
		return errors.New("token, api_id, and api_hash are required")
	}
	if c.SessionFile == "" {
		c.SessionFile = "bot.bbolt"
	}
	return nil
}

func prepareSessionFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		tightenOwnerOnly(dir, 0o700)
	}
	return nil
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
