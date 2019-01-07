# Kit

[![CircleCI](https://circleci.com/gh/thisissoon/gokit.svg?style=svg)](https://circleci.com/gh/thisissoon/gokit)
[![Go Report Card](https://goreportcard.com/badge/go.soon.build/kit)](https://goreportcard.com/report/go.soon.build/kit)

A set of common packages for building applications in Go at SOON_

```
go get go.soon.build/kit
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
