package cache

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Timer represents a timer with its details
type Timer struct {
	TimerID         int64
	RunTimer        int64
	Status          int
	App             string
	NotifyHTTPParam string
	CreateTime      *time.Time
}

// TaskMemCache is a concurrent safe cache for tasks
type TaskMemCache struct {
	tasks      sync.Map
	timerHeap  *TimerHeap
	heapLocker sync.Mutex
}

// NewTaskMemCache creates a new TaskMemCache
func NewTaskMemCache() *TaskMemCache {
	return &TaskMemCache{
		timerHeap: &TimerHeap{},
	}
}

// Set stores a timer in the cache
func (c *TaskMemCache) Set(timer Timer) {
	c.tasks.Store(timer.TimerID, timer)
	c.heapLocker.Lock()
	heap.Push(c.timerHeap, timer)
	c.heapLocker.Unlock()
}

// Get retrieves a timer from the cache
func (c *TaskMemCache) Get(timerID int64) (Timer, bool) {
	value, ok := c.tasks.Load(timerID)
	if ok {
		return value.(Timer), true
	}
	return Timer{}, false
}

// RangeQuery performs a range query on the heap
func (c *TaskMemCache) RangeQuery(start, end int64) ([]Timer, error) {
	c.heapLocker.Lock()
	defer c.heapLocker.Unlock()

	var result []Timer
	for _, timer := range *c.timerHeap {
		if timer.CreateTime.Unix() >= start && timer.CreateTime.Unix() <= end {
			result = append(result, timer)
		}
	}
	if len(result) == 0 {
		return nil, errors.New("no timer found")
	}
	fmt.Print("RangeQuery result: ", result)
	return result, nil
}

// TimerHeap is a min-heap of timers
type TimerHeap []Timer

func (h TimerHeap) Len() int           { return len(h) }
func (h TimerHeap) Less(i, j int) bool { return h[i].CreateTime.Before(*h[j].CreateTime) }
func (h TimerHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TimerHeap) Push(x interface{}) {
	*h = append(*h, x.(Timer))
}

func (h *TimerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
