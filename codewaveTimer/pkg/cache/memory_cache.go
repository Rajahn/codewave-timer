package cache

import (
	"container/heap"
	"sync"
)

// Task represents a task with a TimerID and RunTimer
type Task struct {
	TimerID  int64
	RunTimer int64
}

// TaskCache is a concurrent safe cache for tasks
type TaskCache struct {
	tasks      sync.Map
	taskHeap   *TaskHeap
	heapLocker sync.Mutex
}

// NewTaskCache creates a new TaskCache
func NewTaskCache() *TaskCache {
	return &TaskCache{
		taskHeap: &TaskHeap{},
	}
}

// Set stores a task in the cache
func (c *TaskCache) Set(key int64, value Task) {
	c.tasks.Store(key, value)
	c.heapLocker.Lock()
	heap.Push(c.taskHeap, value)
	c.heapLocker.Unlock()
}

// Get retrieves a task from the cache
func (c *TaskCache) Get(key int64) (Task, bool) {
	value, ok := c.tasks.Load(key)
	if ok {
		return value.(Task), true
	}
	return Task{}, false
}

// RangeQuery performs a range query on the heap
func (c *TaskCache) RangeQuery(start, end int64) []Task {
	c.heapLocker.Lock()
	defer c.heapLocker.Unlock()

	var result []Task
	for _, task := range *c.taskHeap {
		if task.RunTimer >= start && task.RunTimer <= end {
			result = append(result, task)
		}
	}
	return result
}

// TaskHeap is a min-heap of tasks
type TaskHeap []Task

func (h TaskHeap) Len() int           { return len(h) }
func (h TaskHeap) Less(i, j int) bool { return h[i].RunTimer < h[j].RunTimer }
func (h TaskHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *TaskHeap) Push(x interface{}) {
	*h = append(*h, x.(Task))
}

func (h *TaskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
