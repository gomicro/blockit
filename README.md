# Blockit
[![Build Status](https://travis-ci.org/gomicro/blockit.svg)](https://travis-ci.org/gomicro/blockit)
[![Go Reportcard](https://goreportcard.com/badge/github.com/gomicro/blockit)](https://goreportcard.com/report/github.com/gomicro/blockit)
[![GoDoc](https://godoc.org/github.com/gomicro/blockit?status.svg)](https://godoc.org/github.com/gomicro/blockit)
[![License](https://img.shields.io/github/license/gomicro/blockit.svg)](https://github.com/gomicro/blockit/blob/master/LICENSE.md)
[![Release](https://img.shields.io/github/release/gomicro/blockit.svg)](https://github.com/gomicro/blockit/releases/latest)

Blockit provides an interface for establishing blockers where you do not wish to proceed until a certain number of tasks are done. Each of the defined blockers can then be fed into a Multiblocker that will wait for all of them to be finished before proceeding.

# Primary Use Case
The reason this library exists is to allow for configuration of a service to proceed all the way to the point of standing up a status endpoint. It is intended to allow the service to configure itself asynchronously of its dependencies and not report healthy until those dependencies are all met. This is particularly useful with Docker and having the service start before its database, telemetry infrastructure, and logging infrastructure are in place.

# Requirements
Golang version 1.9 or higher

# Installation

```
go get github.com/gomicro/blockit
```

# Usage
See the [examples](https://godoc.org/github.com/gomicro/blockit#pkg-examples) within the docs for ways to use the library.

# Versioning
The library will be versioned in accordance with [Semver 2.0.0](http://semver.org).  See the [releases](https://github.com/gomicro/blockit/releases) section for the latest version.  Until version 1.0.0 the libary is considered to be unstable.

It is always highly recommended to vendor the version you are using.

# License
See [LICENSE.md](./LICENSE.md) for more information.
