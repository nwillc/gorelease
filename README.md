[![license](https://img.shields.io/github/license/nwillc/gorelease.svg)](https://tldrlegal.com/license/-isc-license)
[![CI](https://github.com/nwillc/gorelease/workflows/CI/badge.svg)](https://github.com/nwillc/gorelease/actions?query=workflow%3CI)
[![Go Report Card](https://goreportcard.com/badge/github.com/nwillc/gorelease)](https://goreportcard.com/report/github.com/nwillc/gorelease)
[![GitHub tag](https://img.shields.io/github/tag/nwillc/gorelease.svg)](https://github.com/nwillc/gorelease/releases/latest)
------
# Go Release!

A simple program to handle GitHub releases for Go repositories.  

# Get

```bash
go get github.com/nwillc/gorelease
```

# Setup

Your repository should contain the following:

```text
.version
LICENSE.md (Optional)
```

## Your Semantic Version

The `.version` file should contain the semantic version tag you want to use, for example `v0.1.0`.

## Your Code License

This file is optional, if present the text in this file will be used as a comment for the `version.go` files generated.

# Use

Assuming you've set up as above.

1. Commit your code in preparation for release.
1. Update the `.version` file with a new version number.
1. Run `gorelease`

This will:
 
1. generate a new `gen/version/version.go`
1. Create a tag with the version in `.version`
1. push the tag 
1. push the repository

If the push fails due to credential issues it will inform you how to do the push manually. 

# Using the version in your code

This program will generate a `gen/version/version.go` file like [the one in this repo](./gen/version/version.go).
You can reference `version.Version` in your code to access the version tag of the current release.

# Options

```text
Usage of gorelease:
  -dirty
    	Allow dirty repository with uncommitted files.
  -dry-run
    	Perform a dry run, no files changed or tags/files pushed.
  -output string
    	Where to put the output version.go file (default "gen/version/version.go")
  -version
    	Display version.
```

