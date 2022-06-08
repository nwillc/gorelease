/*
 * Copyright (c) 2022,  nwillc@gmail.com
 *
 * Permission to use, copy, modify, and/or distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 *
 */

package setup

import (
	"flag"
)

// Flags used on the command line.
var Flags struct {
	Dirty   *bool
	DryRun  *bool
	Output  *string
	Verbose *bool
	Version *bool
}

func init() {
	Flags.Dirty = flag.Bool("dirty", false, "Allow Dirty repository with uncommitted files.")
	Flags.DryRun = flag.Bool("dry-run", false, "Perform a dry run, no files changed or tags/files pushed.")
	Flags.Output = flag.String("output", DefaultVersionGo, "Where to put the output version.go file")
	Flags.Verbose = flag.Bool("verbose", false, "Verbose mode, more info on some errors")
	Flags.Version = flag.Bool("version", false, "Display version.")
}
