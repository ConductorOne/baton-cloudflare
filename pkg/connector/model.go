package connector

import "github.com/cloudflare/cloudflare-go"

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
