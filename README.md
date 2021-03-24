# Logmanager

[![Go Reference][gopkg_badge]][gopkg]
[![Go Workflow][go_workflow_badge]][go_workflow]
[![Coverage Status][coverage_badge]][coverage]
[![Go Report][report_badge]][report]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]

---

## Table of Contents

1. [Introduction](#introduction)
1. [Installation](#Installation)
1. [Usage](#usage)
1. [Contributing](#contributing)
1. [License](#license)

## Introduction

_Logmanager_ is yet another Go logging library.

## Installation

### Install using `go get`

```shell
$ go get github.com/axiomhq/logmanager
```

### Install from source

```shell
$ git clone https://github.com/axiomhq/logmanager.git
$ cd logmanager
$ make 
```

## Usage

```go
// Simple console logger
log2console := logmanager.GetLogger("foo.bar")
log2console.Info("hello world")
log2console.Warn("it's a trap")

// Prints:
// [09:15:54.24] info  main@foo.bar main.go:10 hello world
// [09:15:54.24] warn  main@foo.bar main.go:11 it's a trap
```

## Contributing

Feel free to submit PRs or to fill issues. Every kind of help is appreciated. 

Before committing, `make` should run without any issues.

Kindly check our [Contributing](Contributing.md) guide on how to propose
bugfixes and improvements, and submitting pull requests to the project.

## License

&copy; Axiom, Inc., 2021

Distributed under MIT License (`The MIT License`).

See [LICENSE](LICENSE) for more information.

<!-- Badges -->

[gopkg]: https://pkg.go.dev/github.com/axiomhq/logmanager
[gopkg_badge]: https://img.shields.io/badge/doc-reference-007d9c?logo=go&logoColor=white&style=flat-square
[go_workflow]: https://github.com/axiomhq/logmanager/actions?query=workflow%3Ago
[go_workflow_badge]: https://img.shields.io/github/workflow/status/axiomhq/logmanager/go?style=flat-square&ghcache=unused
[coverage]: https://codecov.io/gh/axiomhq/logmanager
[coverage_badge]: https://img.shields.io/codecov/c/github/axiomhq/logmanager.svg?style=flat-square&ghcache=unused
[report]: https://goreportcard.com/report/github.com/axiomhq/logmanager
[report_badge]: https://goreportcard.com/badge/github.com/axiomhq/logmanager?style=flat-square&ghcache=unused
[release]: https://github.com/axiomhq/logmanager/releases/latest
[release_badge]: https://img.shields.io/github/release/axiomhq/logmanager.svg?style=flat-square&ghcache=unused
[license]: https://opensource.org/licenses/MIT
[license_badge]: https://img.shields.io/github/license/axiomhq/logmanager.svg?color=blue&style=flat-square&ghcache=unused
