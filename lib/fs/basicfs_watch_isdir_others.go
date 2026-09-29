// Model-output: Claude Opus 5.5
// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !linux
// +build !linux

package fs

// dirAddedOrRemoved reports whether sys, the watcher's system-specific event
// data, says a directory was created, deleted or moved in or out; that is
// only implemented for inotify.
func dirAddedOrRemoved(interface{}) bool {
	return false
}
