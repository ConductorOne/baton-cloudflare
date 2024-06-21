![Baton Logo](./docs/images/baton-logo.png)

# `baton-cloudflare` [![Go Reference](https://pkg.go.dev/badge/github.com/conductorone/baton-cloudflare.svg)](https://pkg.go.dev/github.com/conductorone/baton-cloudflare) ![main ci](https://github.com/conductorone/baton-cloudflare/actions/workflows/main.yaml/badge.svg)

`baton-cloudflare` is a connector for cloudflare built using the [Baton SDK](https://github.com/conductorone/baton-sdk). It communicates with the cloudflare API to sync data about users and roles.

Check out [Baton](https://github.com/conductorone/baton) to learn more the project in general.

# Getting Started

## brew

```
brew install conductorone/baton/baton conductorone/baton/baton-cloudflare
baton-cloudflare
baton resources
```

## docker

```
docker run --rm -v $(pwd):/out -e BATON_ACCOUNT_ID=accountID -e BATON_API_KEY=apiKey -e BATON_EMAIL_ID=emailId ghcr.io/conductorone/baton-cloudflare:latest -f "/out/sync.c1z"
docker run --rm -v $(pwd):/out ghcr.io/conductorone/baton:latest -f "/out/sync.c1z" resources
```

## source

```
go install github.com/conductorone/baton/cmd/baton@main
go install github.com/conductorone/baton-cloudflare/cmd/baton-cloudflare@main

BATON_ACCOUNT_ID=accountID BATON_API_KEY=apiKey BATON_EMAIL_ID=emailId
baton resources
```

# Data Model

`baton-cloudflare` will pull down information about the following cloudflare resources:
- Users
  - Users supervisors
  - roles

# Contributing, Support and Issues

We started Baton because we were tired of taking screenshots and manually building spreadsheets. We welcome contributions, and ideas, no matter how small -- our goal is to make identity and permissions sprawl less painful for everyone. If you have questions, problems, or ideas: Please open a Github Issue!

See [CONTRIBUTING.md](https://github.com/ConductorOne/baton/blob/main/CONTRIBUTING.md) for more details.

# `baton-cloudflare` Command Line Usage

```
baton-cloudflare

Usage:
  baton-cloudflare [flags]
  baton-cloudflare [command]

Available Commands:
  capabilities       Get connector capabilities
  completion         Generate the autocompletion script for the specified shell
  help               Help about any command

Flags:
      --account-id string      The account id for the Cloudflare account. ($BATON_ACCOUNT_ID)
      --api-key string         The api key for the Cloudflare account. ($BATON_API_KEY)
      --api-token string       The api torkn for the Cloudflare account. ($BATON_API_TOKEN)
      --client-id string       The client ID used to authenticate with ConductorOne ($BATON_CLIENT_ID)
      --client-secret string   The client secret used to authenticate with ConductorOne ($BATON_CLIENT_SECRET)
      --email-id string        The email id for the Cloudflare account. ($BATON_EMAIL_ID)
  -f, --file string            The path to the c1z file to sync with ($BATON_FILE) (default "sync.c1z")
  -h, --help                   help for baton-cloudflare
      --log-format string      The output format for logs: json, console ($BATON_LOG_FORMAT) (default "json")
      --log-level string       The log level: debug, info, warn, error ($BATON_LOG_LEVEL) (default "info")
  -p, --provisioning           This must be set in order for provisioning actions to be enabled. ($BATON_PROVISIONING)
      --ticketing              This must be set to enable ticketing support ($BATON_TICKETING)
  -v, --version                version for baton-cloudflare

Use "baton-cloudflare [command] --help" for more information about a command.

```
