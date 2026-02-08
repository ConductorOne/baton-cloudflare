package connector

import "github.com/cloudflare/cloudflare-go"

type Config struct {
	AccountId string
	ApiToken  string
	EmailId   string
	ApiKey    string
	BaseURL   string
}

type Cloudflare struct {
	client    *cloudflare.API
	accountId string
	emailId   string
}

type Response struct {
	Errors   []cloudflare.ResponseInfo `json:"errors"`
	Messages []cloudflare.ResponseInfo `json:"messages"`
	Success  bool                      `json:"success"`
	Result   cloudflare.AccountMember  `json:"result"`
}

type roles struct {
	ID string
}
