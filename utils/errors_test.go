/*
 * Copyright (c) 2020,  nwillc@gmail.com
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
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCheckIfError(t *testing.T) {

	type args struct {
		msg string
		err error
	}
	tests := []struct {
		name string
		args args
		panics bool
	}{
		{
			name: "NoError",
			args: args{
				msg: "An error occurred",
				err: nil,
			},
			panics: false,
		},
		{
			name: "PanicOnError",
			args: args{
				msg: "An error occurred",
				err: fmt.Errorf(""),
			},
			panics: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ptf assert.PanicTestFunc = func() {
				CheckIfError(tt.args.msg, tt.args.err)
			}
			if tt.panics {
				assert.Panics(t, ptf, "should panic on %s, %v", tt.args.msg, tt.args.err)
			} else {
				assert.NotPanics(t, ptf, "should not panic on %s, %v", tt.args.msg, tt.args.err)
			}
		})
	}
}
