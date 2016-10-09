package project

import (
	"errors"
	"sort"
	"fmt"
)

// Priority The priority of the submit request.
type Priority int

var (
	// PNormal The normal priority that all submit requests get.
	PNormal = P4
	// P4 The normal priority that all submit requests get.
	P4 Priority
	// P3 Priority 3
	P3 Priority = 997
	// P2 Priority 2
	P2 Priority = 998
	// P1 Priority 1
	P1 Priority = 999

	// ErrQueueEmpty The error returned when the queue is empty
	ErrQueueEmpty = errors.New("queue empty")

	// ErrQueueUnsorted The error returned when the queue is unsorted
	ErrQueueUnsorted = errors.New("queue unsorted")
)

// SubmitRequest The data for each submit request
type SubmitRequest struct {
	Priority Priority
	IsEmergency bool
	Project string
	Repo string
	PRNumber int
	FromRef string
	Sha1 string
}

// SubmitQueue The submit queue.
type SubmitQueue struct {
	sorted bool
	items  []SubmitRequest
}

// NewQueue Creates a new queue with the items present.
// The resulting queue has not yet been sorted.
func NewQueue() *SubmitQueue {
	return &SubmitQueue{
		sorted: false,
		items:  nil,
	}
}

// IsSorted Returns true if the queue is already sorted.
func (q *SubmitQueue) IsSorted() bool {
	return q.sorted
}

// Enqueue Adds an item to the queue. The queue must be sorted again.
func (q *SubmitQueue) Enqueue(item SubmitRequest) {
	q.items = append(q.items, item)
	q.sorted = false
}

// Sort Creates a new Queue ordered according to priority rules
func (q *SubmitQueue) Sort() {
	// the order is: emergency -> P1 -> P2 -> P3 -> P4/PNormal;
	// each section is sub ordered by time of enqueue
	sort.Sort(byPriorityDescending(q.items))
	sort.Stable(byEmergencyFirst(q.items))
	q.sorted = true
}

// Dequeue Dequeues an item from a sorted queue. Queue must be resorted.
func (q *SubmitQueue) Dequeue() (SubmitRequest, error) {
	if len(q.items) == 0 {
		return SubmitRequest{}, ErrQueueEmpty
	}
	if !q.sorted {
		return SubmitRequest{}, ErrQueueUnsorted
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, nil
}

// Peek Peeks an item from a sorted queue. Queue must be resorted.
func (q *SubmitQueue) Peek() (SubmitRequest, error) {
	if len(q.items) < 1 {
		return SubmitRequest{}, ErrQueueEmpty
	}
	if !q.sorted {
		return SubmitRequest{}, ErrQueueUnsorted
	}
	item := q.items[0]
	return item, nil
}

// Dump dumps the sorted contents of the queue to the io.Writer
func (q *SubmitQueue) Dump() {
	for i, item := range q.items {
		fmt.Printf("[%2d] %s:%d %s\n", i, item.Repo, item.PRNumber, item.Sha1)
	}
}

type byPriorityDescending []SubmitRequest

func (p byPriorityDescending) Len() int {
	return len(p)
}
func (p byPriorityDescending) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p byPriorityDescending) Less(i, j int) bool {
	return p[i].Priority > p[j].Priority
}

type byEmergencyFirst []SubmitRequest

func (p byEmergencyFirst) Len() int {
	return len(p)
}
func (p byEmergencyFirst) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p byEmergencyFirst) Less(i, j int) bool {
	return p[i].IsEmergency && !p[j].IsEmergency
}
