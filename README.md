# Kit

[![CircleCI](https://circleci.com/gh/thisissoon/gokit.svg?style=svg)](https://circleci.com/gh/thisissoon/gokit)
[![Go Report Card](https://goreportcard.com/badge/go.soon.build/kit)](https://goreportcard.com/report/go.soon.build/kit)

A set of common packages for building applications in Go at SOON_.
The packages are split into modules to enable granular control
over dependencies.

## Modules

### Config
Common configuration management with [viper](https://github.com/spf13/viper). Supports toml files, auto ENV var bindings and cobra command flag overrides.
```
go get go.soon.build/kit/config
```

### gRPC
Common helper constructs for running a gRPC server.
```
go get go.soon.build/kit/grpc
```

### HTTP
Common helper constructs for running a HTTP server, using the `http.Handler` pattern from the standard library.
```
go get go.soon.build/kit/http
```

### PSQL
Common helpers for managing postgres database connections and migrations.
```
go get go.soon.build/kit/psql
```

### PubSub
A super minimal publish/subscribe interface with backend implementations for different providers:
 - Google Cloud PubSub
```
go get go.soon.build/kit/pubsub
```

## Development

### Tests

To run the test suite with coverage report:
```
make test
```

To run pubsub tests with gcloud emulator:
```bash
❯ gcloud beta emulators pubsub start
...
❯ $(gcloud beta emulators pubsub env-init)
❯ make testgcloud
```
