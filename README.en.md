# vsixctl

[![codecov](https://codecov.io/gh/E-n-d-l-e-s-s-A-I/vsixctl/branch/master/graph/badge.svg)](https://codecov.io/gh/E-n-d-l-e-s-s-A-I/vsixctl)

Fast and reliable asynchronous CLI extension manager for VS Code

[![CI](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/actions/workflows/ci.yml/badge.svg)](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/E-n-d-l-e-s-s-A-I/vsixctl/branch/master/graph/badge.svg)](https://codecov.io/gh/E-n-d-l-e-s-s-A-I/vsixctl)
[![Release](https://img.shields.io/github/v/release/E-n-d-l-e-s-s-A-I/vsixctl)](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/E-n-d-l-e-s-s-A-I/vsixctl)](https://goreportcard.com/report/github.com/E-n-d-l-e-s-s-A-I/vsixctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE) 

[Русский](README.md)

## Features

- Search extensions on VS Code Marketplace
- Asynchronous installation, updating and removal
- Install specific extension versions

## Installation

### Linux (WSL)

```sh
curl -sSL https://raw.githubusercontent.com/E-n-d-l-e-s-s-A-I/vsixctl/master/install.sh | sh
```

### Windows

Download a zip archive for your platform from the [Releases](https://github.com/E-n-d-l-e-s-s-A-I/vsixctl/releases) page, extract it and add the directory containing `vsixctl.exe` to your `PATH` environment variable.

### From source

Requires [Go](https://go.dev/dl/) 1.25+.

```sh
go install github.com/E-n-d-l-e-s-s-A-I/vsixctl@latest
```

## Usage

```sh
# Full-text search
vsixctl search go

# Search by id
vsixctl search golang.go --type id

# Search by name
vsixctl search go --type name

# Install an extension
vsixctl install golang.go ms-python.python

# Install a specific version
vsixctl install golang.go@0.44.0

# List installed extensions
vsixctl ls

# Available versions of an extension
vsixctl versions golang.go

# Update all extensions
vsixctl update

# Update specific extensions
vsixctl update golang.go esbenp.prettier-vscode

# Remove an extension
vsixctl rm golang.go
```
