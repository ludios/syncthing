// Model-output: Claude Opus 5.5
// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build linux

package fs

import (
	"errors"
	"syscall"
)

func reachedMaxUserWatches(err error) bool {
	var errno syscall.Errno
	return errors.As(err, &errno) && (errno == syscall.EMFILE || errno == syscall.ENOSPC)
}
