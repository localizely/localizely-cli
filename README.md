# Localizely CLI

The Localizely CLI helps you sync files between your project and the [Localizely](https://localizely.com/) platform.

[Localizely](https://localizely.com/) is a translation management platform that helps you translate texts in your app for targeting multilingual market.

## Install

The Localizely CLI can be installed on MacOS, Linux, and Windows.

### Downlaod from Github Releases

[Download](https://github.com/localizely/localizely-cli/releases) the binary for your platform and place the executable in the path. That's it.

### Build from source

Clone the repo and build from the source.

_Note: This requires you to have Go installed on your system._

## Usage

List available commands:

```bash
localizely-cli --help
```

### Commands

Below you can find a brief description of the available commands.

#### Init

Configure your Localizely client.

```bash
localizely-cli init
```

#### Pull

Pull localization files from Localizely.

```bash
localizely-cli pull
```

#### Push

Push localization files to Localizely.

```bash
localizely-cli push
```

#### Update

Update Localizely CLI to the latest version.

```bash
localizely-cli update
```
