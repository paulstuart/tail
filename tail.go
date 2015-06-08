// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !plan9,!solaris

package tail

import (
	"bufio"
	"io"
	"log"
	"os"
	"path"

	"github.com/go-fsnotify/fsnotify"
)

type Tail struct {
	*os.File
	Last   int64
	Feed   chan []byte
	b      []byte
	follow bool
}

func (t *Tail) Lines() chan string {
	c := make(chan string)
	go func() {
		s := bufio.NewScanner(t)
		for s.Scan() {
			c <- s.Text()
		}
		close(c)
	}()
	return c
}

func (t *Tail) Write() {
	if t == nil || t.File == nil {
		return
	}

	fi, err := t.File.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if fi.Size() < t.Last {
		t.Last = 0
	}

	for {
		if fi.Size() == t.Last {
			return
		}

		n, err := t.File.ReadAt(t.b, t.Last)
		if err != nil && err != io.EOF {
			log.Println("ERR:", err)
		}
		if n == 0 {
			break
		}
		t.Last += int64(n)
		t.Feed <- t.b[:n]
	}
}

func (t *Tail) Read(b []byte) (int, error) {
	n := copy(b, <-t.Feed)
	if t.File == nil && !t.follow {
		return n, io.EOF
	}
	return n, nil
}

func NewTail(file string, follow bool) *Tail {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	t := &Tail{Feed: make(chan []byte, 8192), b: make([]byte, 8192), follow: follow}

	if err = watcher.Add(file); err != nil {
		log.Fatal(err)
	}

	if t.File, err = os.Open(file); err != nil {
		log.Fatal(err)
	}

	if err = watcher.Add(path.Dir(file)); err != nil {
		log.Fatal(err)
	}

	go func() {
		defer watcher.Close()
		t.Write() // for whatever is already there
		for {
			select {
			case event := <-watcher.Events:
				switch {
				case event.Op&fsnotify.Write == fsnotify.Write:
					t.Write()
				case event.Op&fsnotify.Create == fsnotify.Create:
					if t.File == nil && file == path.Base(event.Name) {
						if t.File, err = os.Open(file); err != nil {
							log.Fatal("reopen:", err)
						}
						if err = watcher.Add(file); err != nil {
							log.Fatal("add err:", err)
						}
					}
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					if t.File != nil && t.File.Name() == path.Base(event.Name) {
						t.File.Close()
						t.File = nil
						if !follow {
							close(t.Feed)
							goto DONE
						}
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	DONE:
	}()

	return t
}
