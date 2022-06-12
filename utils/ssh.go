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

package utils

import (
	"github.com/nwillc/genfuncs"
	"os"
	"os/user"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/nwillc/gorelease/setup"
)

// PublicKeys returns current users public keys.
func PublicKeys() *genfuncs.Result[*ssh.PublicKeys] {
	path, err := os.UserHomeDir()
	CheckIfError("finding home directory", err)
	path += "/.ssh/id_rsa"

	return genfuncs.NewResultFromTuple(ssh.NewPublicKeysFromFile(setup.GitUser, path, ""))
}

// NewSignature create a minimal object.Signature for the current user.
func NewSignature() *object.Signature {
	r := genfuncs.NewResultFromTuple(user.Current())
	sig := object.Signature{
		Name: r.MustGet().Name,
		When: time.Now(),
	}
	return &sig
}
