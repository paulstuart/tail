// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !plan9,!solaris

package tail

import (
	"fmt"
	//"log"
	"os"
	"testing"
	"time"
)

var (
	sample = "test.txt"
	quit   = make(chan struct{})
)

func testFile() *os.File {
	file, err := os.OpenFile(sample, os.O_RDWR, 0666)
	if err != nil {
		file, err = os.Create(sample)
		if err != nil {
			panic(err)
		}
	}
	return file
}

func ticker(limit int, redo bool) {
	tick := time.NewTicker(1 * time.Second).C
	file := testFile()
	go func() {
		cnt := 0
		for {
			select {
			case <-tick:
				//fmt.Println("TICK:", cnt, time.Now())
				fmt.Fprintln(file, cnt, time.Now())
				file.Sync()
				cnt++
				if cnt < limit {
					continue
				}
				file.Close()
				os.Remove(sample)
				//t.Log("DELETED SAMPLE FILE")
				if !redo {
					//fmt.Println("BREAK SAMPLE")
					goto DONE
				}
				file = testFile()
				cnt = 0
			case <-quit:
				break
			}
		}
	DONE:
		//fmt.Println("END OF SAMPLE")
	}()
}

func TestStrings(t *testing.T) {
	ticker(3, false)
	w := NewTail(sample, false)
	for s := range w.Strings() {
		t.Log("TAIL:", s)
	}
}

func TestResume(t *testing.T) {
	ticker(3, true)
	w := NewTail(sample, true)
	c := 0
	for s := range w.Strings() {
		t.Log("TAIL:", s)
		c++
		if c > 10 {
			break
		}
	}
}
