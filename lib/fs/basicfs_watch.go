// Model-output: Claude Opus 5.5
// Copyright (C) 2016 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

//go:build !(solaris && !cgo) && !(darwin && !cgo) && !(darwin && kqueue)
// +build !solaris cgo
// +build !darwin cgo
// +build !darwin !kqueue

package fs

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/syncthing/notify"
)

// Notify does not block on sending to channel, so the channel must be buffered.
// The actual number is magic.
// Not meant to be changed, but must be changeable for tests
var backendBuffer = 500

func (f *BasicFilesystem) Watch(name string, ignore Matcher, ctx context.Context, ignorePerms bool) (<-chan Event, <-chan error, error) {
	watchPath, roots, err := f.watchPaths(name)
	if err != nil {
		return nil, nil, err
	}

	outChan := make(chan Event)
	backendChan := make(chan notify.EventInfo, backendBuffer)

	eventMask := subEventMask
	if !ignorePerms {
		eventMask |= permEventMask
	}

	absShouldIgnore := func(absPath string) bool {
		if !utf8.ValidString(absPath) {
			return true
		}

		rel, err := f.unrootedChecked(absPath, roots)
		if err != nil {
			return true
		}
		return ignore.Match(rel).CanSkipDir()
	}
	err = notify.WatchWithFilter(watchPath, backendChan, absShouldIgnore, eventMask)
	if err != nil {
		notify.Stop(backendChan)
		if reachedMaxUserWatches(err) {
			err = errors.New("failed to set up inotify handler. Please increase inotify limits, see https://docs.syncthing.net/users/faq.html#inotify-limits")
		}
		return nil, nil, err
	}

	errChan := make(chan error)
	go f.watchLoop(ctx, name, roots, backendChan, outChan, errChan, ignore)

	return outChan, errChan, nil
}

// enclosingGitDir returns the path of the outermost .git in name (the git
// directory, or a gitfile pointing to one), or name itself if that is .git,
// and whether there is one. Bare repositories (foo.git) are not recognised.
func enclosingGitDir(name string) (string, bool) {
	slashed := "/" + filepath.ToSlash(name) + "/"
	i := strings.Index(slashed, "/.git/")
	if i < 0 {
		return "", false
	}
	return filepath.FromSlash(slashed[1 : i+len("/.git")]), true
}

// watchEvent returns the event to report for relPath changing, as ev says.
//
// Directories are created, removed and renamed inside git directories all the
// time (gc prunes .git/objects/xx, commits recreate them), and changes around
// that are what the watcher has missed. So a directory appearing or going away
// inside a git directory is reported as a change of the whole git directory:
// it gets rescanned shortly after, finding whatever else changed there
// unreported.
func (f *BasicFilesystem) watchEvent(relPath string, ev notify.EventInfo) Event {
	if dirAddedOrRemoved(ev.Sys()) {
		if gitDir, ok := enclosingGitDir(relPath); ok {
			l.Debugln(f.Type(), f.URI(), "Watch: Directory", relPath, "added or removed, rescanning", gitDir)
			return Event{Name: gitDir, Type: NonRemove}
		}
	}
	return Event{Name: relPath, Type: f.eventType(ev.Event())}
}

func (f *BasicFilesystem) watchLoop(ctx context.Context, name string, roots []string, backendChan chan notify.EventInfo, outChan chan<- Event, errChan chan<- error, ignore Matcher) {
	for {
		// Detect channel overflow
		if len(backendChan) == backendBuffer {
		outer:
			for {
				select {
				case <-backendChan:
				default:
					break outer
				}
			}
			// When next scheduling a scan, do it on the entire folder as events have been lost.
			outChan <- Event{Name: name, Type: NonRemove}
			l.Debugln(f.Type(), f.URI(), "Watch: Event overflow, send \".\"")
		}

		select {
		case ev := <-backendChan:
			evPath := ev.Path()

			if !utf8.ValidString(evPath) {
				l.Debugln(f.Type(), f.URI(), "Watch: Ignoring invalid UTF-8")
				continue
			}

			relPath, err := f.unrootedChecked(evPath, roots)
			if err != nil {
				select {
				case errChan <- err:
					l.Debugln(f.Type(), f.URI(), "Watch: Sending error", err)
				case <-ctx.Done():
				}
				notify.Stop(backendChan)
				l.Debugln(f.Type(), f.URI(), "Watch: Stopped due to", err)
				return
			}

			if ignore.Match(relPath).IsIgnored() {
				l.Debugln(f.Type(), f.URI(), "Watch: Ignoring", relPath)
				continue
			}
			event := f.watchEvent(relPath, ev)
			select {
			case outChan <- event:
				l.Debugln(f.Type(), f.URI(), "Watch: Sending", event.Name, event.Type)
			case <-ctx.Done():
				notify.Stop(backendChan)
				l.Debugln(f.Type(), f.URI(), "Watch: Stopped")
				return
			}
		case <-ctx.Done():
			notify.Stop(backendChan)
			l.Debugln(f.Type(), f.URI(), "Watch: Stopped")
			return
		}
	}
}

func (*BasicFilesystem) eventType(notifyType notify.Event) EventType {
	if notifyType&rmEventMask != 0 {
		return Remove
	}
	return NonRemove
}
