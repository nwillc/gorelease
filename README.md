[![license](https://img.shields.io/github/license/nwillc/gorelease.svg)](https://tldrlegal.com/license/-isc-license)
[![CI](https://github.com/nwillc/gorelease/workflows/CI/badge.svg)](https://github.com/nwillc/gorelease/actions?query=workflow%3CI)
[![Go Report Card](https://goreportcard.com/badge/github.com/nwillc/gorelease)](https://goreportcard.com/report/github.com/nwillc/gorelease)
------
# Go Release!

A simple program to handle GitHub releases for Go repositories.  

# Get

```bash
go get github.com/nwillc/gorelease
go install
```

# Setup

Your repository should contain the following:

```text
.version
LICENSE.md
version/version.go
```

## Your Semantic Version

The `.version` file should contain the semantic version you want to use, for example `0.1.0`.

## Your Code License

The test in this file will be used as a comment for the `version.go` files generated.

## Your Version Go File

This can be empty to start, `gorelease` will create a valid Go file basically containing:

```go
package version

var Version = "v0.1.0"
```

This can be referenced in your Go.

# Use

Assuming you've set up as above.

1. Commit your code in preparation for release.
1. Update the `.version` file with a new version number.
1. Run `gorelease`, or use a `//go:generate gorelease` in your code and `go generate` to run it.

This will:
 
1. generate a new `version/version.go`
1. Create a tag with the version, prepended by a `v`
1. push the tag 
1. push the repository

If the push fails due to credential issues it will inform you how to do the push manually. 
