# Localizely CLI

The [Localizely CLI](https://localizely.com/cli/) is a command line tool that helps you sync localization files between your project and the Localizely platform.

[Localizely](https://localizely.com/) is a translation management platform that helps you translate texts in your app for targeting multilingual market.

## Install

The Localizely CLI can be installed on all major platforms (MacOS, Linux, and Windows).  
You can find more details about the installation below.

### Download from GitHub Releases

[Download](https://github.com/localizely/localizely-cli/releases) the binary for your platform, unzip it, and place the executable in the path. That's it.

### Docker

Run the Localizely CLI from Docker.

```bash
docker run --rm localizely/localizely-cli
```

_**Note:** This requires you to have Docker installed on your system._

### Build from source

Clone the repo and build from the source.

Clone the repository

```bash
git clone https://github.com/localizely/localizely-cli.git
```

Move to the project directory

```bash
cd localizely-cli
```

Download project dependencies

```bash
go mod download
```

Build the project

```bash
go build
```

After successfully completing the above steps, you should see the executable for your platform in the root of the project.

_**Note:** This requires you to have Go installed on your system._

## Usage

Below you can find a brief description of the available commands.

### Init

Configure your Localizely client.

This command offers two modes: interactive and template. The first one will guide you through the configuration of your Localizely client. The second one will generate a template file that needs to be updated with your config data.  
After executing this command, a [configuration file](https://localizely.com/configuration-file/) (`localizely.yml`) will be created for you.

The interactive mode

```bash
localizely-cli init
```

The template mode

```bash
localizely-cli init --mode template
```

_**Note:** API token entered through interactive mode is saved in the ~/.localizely/credentials.yaml file._

### Pull

Pull localization files from Localizely.

Depending on your workflow, you can use this command by relying on the external configuration (the `localizely.yml` file), or by passing your configuration through flags.

Rely on the external configuration

```bash
localizely-cli pull
```

Pass configuration through flags

```bash
localizely-cli pull \
  --api-token 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --project-id 01234567-abcd-abcd-abcd-0123456789ab \
  --file-type json \
  --files "file[0]=lang/en.json","locale_code[0]=en","file[1]=lang/de_DE.json","locale_code[1]=de-DE" \
  --export-empty-as empty \
  --include-tags new,updated \
  --exclude-tags removed
```

### Push

Push localization files to Localizely.

Depending on your workflow, you can use this command by relying on the external configuration (the `localizely.yml` file), or by passing your configuration through flags.

Rely on the external configuration

```bash
localizely-cli push
```

Pass configuration through flags

```bash
localizely-cli push \
  --api-token 0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  --project-id 01234567-abcd-abcd-abcd-0123456789ab \
  --files "file[0]=lang/en.json","locale_code[0]=en","file[1]=lang/de_DE.json","locale_code[1]=de-DE" \
  --overwrite \
  --reviewed=false \
  --tag-added new,new-feat-x \
  --tag-updated updated,updated-feat-x \
  --tag-removed removed
```

### Update

Update Localizely CLI to the latest version.

This command checks if there is a newer version of the `localizely-cli` available. If there is, you will be prompted to confirm the update.

```bash
localizely-cli update
```

## Contributing

If anything feels off, or you would like to propose some functionality, feel free to do it through [GitHub Issue Tracker](https://github.com/localizely/localizely-cli/issues).

## Development

The following sections are intended for maintainers of the Localizely CLI.

### API client

All communication with the Localizely API goes through the [localizely-client-go](https://github.com/localizely/localizely-client-go) package, a Go client generated with [OpenAPI Generator](https://openapi-generator.tech/) from the [Localizely OpenAPI specification](https://api.localizely.com/api-docs). Do not add hand-written API calls to this project. When the API changes, regenerate the client, release a new version of it, and update the dependency here.

The client repository is generated as a whole, including its README, so the regeneration process is documented here. The only things you need are the OpenAPI Generator CLI and the `generate.sh` script from that repository.

Install OpenAPI Generator (see the [installation guide](https://openapi-generator.tech/docs/installation) for other options)

```bash
brew install openapi-generator
```

Clone the client repository

```bash
git clone https://github.com/localizely/localizely-client-go.git
cd localizely-client-go
```

Run the script with the new package version as the only argument. It downloads the latest specification from the Localizely API, regenerates all files, and runs `go mod tidy`.

```bash
./generate.sh 1.2.0
```

Review the changes, commit them, and push them together with a tag for the new version

```bash
git tag v1.2.0
git push origin main v1.2.0
```

Update the dependency in this project

```bash
go get github.com/localizely/localizely-client-go@v1.2.0
go mod tidy
```

_**Note:** The script calls the `openapi-generator` executable, which is what the Homebrew package provides. Installing the generator through npm gives you `openapi-generator-cli` instead, so make sure `openapi-generator` is available on your path._

### Releasing a new version

Releases are built and published with [GoReleaser](https://goreleaser.com/), using the configuration in the `.goreleaser.yaml` file. A release creates a [GitHub release](https://github.com/localizely/localizely-cli/releases) with binaries for Linux, macOS, and Windows, and builds and pushes the [localizely/localizely-cli](https://hub.docker.com/r/localizely/localizely-cli) Docker image, tagged with the version and `latest`, to Docker Hub.

Before you start, make sure you have:

- [GoReleaser](https://goreleaser.com/install/) v2 installed
- Docker running and logged in to Docker Hub (`docker login`) with an account that can push to the `localizely/localizely-cli` repository
- A GitHub personal access token that can create releases in this repository (the `repo` scope for a classic token, or the `contents: write` permission for a fine-grained token)

Bump the `Version` constant in `cmd/root.go` to the new version and commit the change. The `update` command compares this constant with the latest GitHub release, so it must match the tag you are about to create.

Create a tag for the new version, prefixed with `v`, and push the commit together with the tag

```bash
git tag v1.0.11
git push origin main v1.0.11
```

Export your GitHub token

```bash
export GITHUB_TOKEN=<your-token>
```

Optionally, do a dry run that builds everything locally without publishing anything

```bash
goreleaser release --snapshot --clean
```

Publish the release

```bash
goreleaser release --clean
```

When the command finishes, verify the [GitHub release](https://github.com/localizely/localizely-cli/releases) and the [Docker Hub tags](https://hub.docker.com/r/localizely/localizely-cli/tags), and confirm that `localizely-cli update` detects the new version.

_**Note:** GoReleaser refuses to release from a working tree with uncommitted changes, or from a commit that is not tagged._

## Useful links

- [Localizely CLI Docs](https://localizely.com/cli/)
- [Localizely API Docs](https://api.localizely.com/swagger-ui/index.html)
- [Localizely Go API client](https://github.com/localizely/localizely-client-go)
- [Localizely Configuration File](https://localizely.com/configuration-file/)
