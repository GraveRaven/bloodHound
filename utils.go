package main

import (
	"container/list"
	"errors"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

func testErrLog(s string, err error) bool {
	if err == nil {
		return false
	}

	log.Printf("%s: %v\n", s, err)
	return true
}

func testErrDie(s string, err error) {
	if err == nil {
		return
	}

	log.Fatalf("%s: %v\n", s, err)
}

type QueueMutex struct {
	sync.Mutex
	list *list.List
}

func NewQueueMutex() (queue *QueueMutex) {
	queue = new(QueueMutex)
	queue.list = list.New()
	return queue
}

func (q *QueueMutex) Push(v interface{}) {
	q.Lock()
	q.list.PushBack(v)
	q.Unlock()
}

func (q *QueueMutex) Pop() interface{} {
	var v *list.Element
	delay := config.WaitDelay //delay in seconds

	for i := 0; i < delay*2; i++ {
		if i == (delay*2)-1 {
			return nil
		}
		q.Lock()
		v = q.list.Front()
		if v == nil {
			q.Unlock()
			time.Sleep(500 * time.Millisecond)
		} else {
			break
		}
	}

	q.list.Remove(v)
	q.Unlock()
	return v.Value
}

func (q *QueueMutex) Len() int {
	return q.list.Len()
}

const (
	BYTE     = 1.0
	KILOBYTE = 1024 * BYTE
	MEGABYTE = 1024 * KILOBYTE
	GIGABYTE = 1024 * MEGABYTE
	TERABYTE = 1024 * GIGABYTE
)

var bytesPattern *regexp.Regexp = regexp.MustCompile(`(?i)^(-?\d+)([KMGT]B?|B)$`)
var invalidByteQuantityError = errors.New("Byte quantity must be a positive integer with a unit of measurement like M, MB, G, or GB")

func ToBytes(s string) (int64, error) {
	parts := bytesPattern.FindStringSubmatch(strings.TrimSpace(s))
	if len(parts) < 3 {
		return 0, invalidByteQuantityError
	}

	value, err := strconv.ParseInt(parts[1], 10, 0)
	if err != nil || value < 1 {
		return 0, invalidByteQuantityError
	}

	var bytes int64
	unit := strings.ToUpper(parts[2])
	switch unit[:1] {
	case "T":
		bytes = value * TERABYTE
	case "G":
		bytes = value * GIGABYTE
	case "M":
		bytes = value * MEGABYTE
	case "K":
		bytes = value * KILOBYTE
	case "B":
		bytes = value * BYTE
	}

	return bytes, nil
}
