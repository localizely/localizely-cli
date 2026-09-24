# Localizely CLI

Go CLI that syncs localization files between a project and the [Localizely](https://localizely.com/) platform. See README.md for install and usage.

CLI commands:

- `init`: writes `localizely.yml` and, in interactive mode, saves the API token to `~/.localizely/credentials.yaml`
- `pull`: download localization files
- `push`: upload localization files
- `update`: self-update from GitHub Releases

## Development

- Build: `go build`
- Check: `go vet ./...` and `gofmt -l .`
- Tests: none yet

## API client

All API calls go through `github.com/localizely/localizely-client-go`, generated with openapi-generator from the OpenAPI spec at https://api.localizely.com/api-docs. Never hand-write HTTP calls; when the API changes, regenerate the client and bump it in `go.mod`.

## Conventions

- Every Go file starts with the MIT license header used by the existing files.
- Bind flags with `viper.BindPFlag` in a command's `PreRun`, not in `init()`: `pull` and `push` share flag names on one global Viper instance.
- Supported file types are listed in `fileTypesOpt` in `cmd/root.go` and repeated in the `init` YAML templates; change them together.

## Releases

- `Version` in `cmd/root.go` is hardcoded; bump it before tagging a release, because `update` compares it with the latest GitHub release.
- Releases are published with `goreleaser release` from `.goreleaser.yaml`. It needs `GITHUB_TOKEN` in the environment, and Docker running and logged in to Docker Hub, because it also builds and pushes the `localizely/localizely-cli` image.
