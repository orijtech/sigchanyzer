# sigchanyzer

![Build status](https://github.com/orijtech/sigchanyzer/workflows/Go/badge.svg?branch=master)

Package sigchanyzer defines an [Analyzer](analyzer_link) that checks usage of unbuffered os.Signal channel, which can be at
risk of missing the signal.

## Installation

With Go modules:

```sh
go get github.com/orijtech/sigchanyzer/cmd/sigchanyzer
```

Without Go modules:

```sh
$ cd $GOPATH/src/github.com/orijtech/sigchanyzer
$ git checkout v0.0.2
$ go get
$ install ./cmd/sigchanyzer
```

## Usage

You can run `sigchanyzer` either on a Go package or Go files, the same way as
other Go tools work.

Example:

```sh
$ sigchanyzer github.com/orijtech/sigchanyzer/testdata/src/a
```

or:

```sh
$ sigchanyzer ./testdata/src/a/a.go
```

Sample output:

```text
/go/src/github.com/orijtech/sigchanyzer/testdata/a/a.go:16:7: unbuffered os.Signal channel
/go/src/github.com/orijtech/sigchanyzer/testdata/a/a.go:22:7: unbuffered os.Signal channel
```
 
## Development

Go 1.15+

### Running test

Add test case to `testdata/src/a` directory, then run:

```shell script
go test
```

## Contributing

All contributions are welcome, please report bug or open a pull request.

[analyzer_link]: https://pkg.go.dev/golang.org/x/tools/go/analysis#Analyzer
