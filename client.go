package main

import (
	"github.com/go-resty/resty/v2"
	"time"
)

var Client *resty.Client

func SetupClient() {
	Client = resty.New().SetTimeout(120 * time.Second)
}
