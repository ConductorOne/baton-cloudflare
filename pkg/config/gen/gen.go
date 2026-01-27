package main

import (
	cfg "github.com/conductorone/baton-cloudflare/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("cloudflare", cfg.Config)
}
