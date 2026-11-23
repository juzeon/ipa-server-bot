package main

import (
	"time"

	"github.com/go-resty/resty/v2"
)

var Client *resty.Client

func SetupClient() {
	Client = resty.New().
		SetTimeout(120 * time.Second).
		SetRedirectPolicy(resty.RedirectPolicyFunc(sameOriginRedirect))
}
