package config

import (
	"flag"
)

const (
	DotVersionFile   = ".version"
	LicenseFile      = "LICENSE.md"
	GitUser          = "git"
	DefaultVersionGo = "gen/version/version.go"
)

var Flags struct {
	DryRun  *bool
	Dirty   *bool
	Version *bool
	Output  *string
}

func init() {
	Flags.DryRun = flag.Bool("dryrun", false, "Perform a dry run, no files changed or tags/files pushed.")
	Flags.Dirty = flag.Bool("dirty", false, "Allow Dirty repository with uncommitted files.")
	Flags.Version = flag.Bool("version", false, "Display version.")
	Flags.Output = flag.String("output", DefaultVersionGo, "Where to put the output version.go file")
}
