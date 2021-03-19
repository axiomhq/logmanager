# logmanager
Yet another golang log manager library

## Table of Contents
- [logmanager](#logmanager)
  - [Table of Contents](#table-of-contents)
  - [Introduction](#introduction)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Contributing](#contributing)
  - [License](#license)

## Introduction

logmanager is a library that writes to standard error and adds a timestamp without the need for configuration.

## Installation

`go get -u github.com/axiomhq/logmanager`

## Usage

```go
// Simple console logger
log2console := logmanager.GetLogger("foo.bar")
log2console.Info("hello world")
log2console.Warn("it's a trap")

// Prints
/*
[09:15:54.24] info  main@foo.bar main.go:10 hello world
[09:15:54.24] warn  main@foo.bar main.go:11 it's a trap
*/
```

## Contributing

Feel free to submit PRs or to fill issues. Every kind of help is appreciated.

Before committing, make should run without any issues.

Kindly check our Contributing guide on how to propose bugfixes and improvements, and submitting pull requests to the project.

## License

&copy; Axiom, Inc., 2021

Distributed under MIT License (`The MIT License`).

See [LICENSE](LICENSE) for more information.

[![License Status][license_status_badge]][license_status]


<!-- Badges -->


