// Model-output: Claude Opus 5.5
// Copyright (C) 2026 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build linux
// +build linux

package fs

import "golang.org/x/sys/unix"

// dirAddedOrRemoved reports whether sys, the watcher's system-specific event
// data, says a directory was created, deleted or moved in or out. inotify says
// so even for a directory that is gone. Events about a watch itself (deleted
// or moved) never carry IN_ISDIR, but only directories are watched.
func dirAddedOrRemoved(sys interface{}) bool {
	ev, ok := sys.(*unix.InotifyEvent)
	if !ok {
		return false
	}
	const entry = unix.IN_CREATE | unix.IN_DELETE | unix.IN_MOVED_FROM | unix.IN_MOVED_TO
	return ev.Mask&unix.IN_ISDIR != 0 && ev.Mask&entry != 0 ||
		ev.Mask&(unix.IN_DELETE_SELF|unix.IN_MOVE_SELF) != 0
}
