# logmanager [![Go Reference][gopkg_badge]][gopkg] [![Workflow][workflow_badge]][workflow] [![Latest Release][release_badge]][release] [![License][license_badge]][license]

Yet another Go logging library with a focus on simplicity and flexibility.

## Install

```shell
go get github.com/axiomhq/logmanager
```

## Usage

```go
package main

import (
    "github.com/axiomhq/logmanager"
)

func main() {
    // Simple console logger
    log := logmanager.GetLogger("foo.bar")
    log.Info("hello world")
    log.Warn("it's a trap")

    // Output:
    // [09:15:54.24] info  main@foo.bar main.go:10 hello world
    // [09:15:54.24] warn  main@foo.bar main.go:11 it's a trap
}
```

## Features

- Multiple log levels (Trace, Debug, Info, Warning, Error, Critical)
- Colored console output with automatic color assignment per module
- File-based logging with automatic rotation
- Syslog support (RFC 5424)
- Thread-safe operations
- Module-based logging with inheritance
- Stack trace support for errors and panics
- Zero dependencies for core functionality

## Configuration

Set log levels via environment variable:

```shell
# Set specific module log levels
export LOGMANAGER_SPEC="foo.bar=Debug:foo=Trace:Info"

# Set global log level
export LOGMANAGER_SPEC="Debug"
```

## Writers

logmanager supports multiple output writers:

### Console Writer

Outputs colored logs to stdout/stderr with automatic color assignment per module.

```go
writer := logmanager.NewConsoleWriter()
logmanager.AddGlobalWriter(writer)
```

### Disk Writer

Writes logs to files with automatic rotation support.

```go
writer := logmanager.NewDiskWriter("/var/log/app.log", logmanager.DiskWriterConfig{
    RotateDuration:  24 * time.Hour,
    MaximumLogFiles: 7,
})
logmanager.AddGlobalWriter(writer)
```

### Syslog Writer

Sends logs to syslog (RFC 5424 format).

```go
writer := logmanager.NewSyslogWriter("myapp", "127.0.0.1:514")
logmanager.AddGlobalWriter(writer)
```

## License

[MIT](LICENSE)

<!-- Badges -->

[gopkg]: https://pkg.go.dev/github.com/axiomhq/logmanager
[gopkg_badge]: https://pkg.go.dev/badge/github.com/axiomhq/logmanager.svg
[workflow]: https://github.com/axiomhq/logmanager/actions/workflows/push.yaml
[workflow_badge]: https://img.shields.io/github/actions/workflow/status/axiomhq/logmanager/push.yaml?branch=main&ghcache=unused
[release]: https://github.com/axiomhq/logmanager/releases/latest
[release_badge]: https://img.shields.io/github/v/release/axiomhq/logmanager?ghcache=unused
[license]: LICENSE
[license_badge]: https://img.shields.io/github/license/axiomhq/logmanager?ghcache=unused
