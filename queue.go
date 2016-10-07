package submitqueue

import (
	"errors"
	"sort"
	"github.com/saquibmian/submitqueue/scm"
	"github.com/saquibmian/submitqueue/project"
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
type SubmitRequest interface {
	Sha1() string
	Priority() Priority
	IsEmergency() bool
	GetProject() project.Project
	GetRepo() scm.Repo
	GetPR() scm.PullRequest
}

// SubmitQueue The submit queue.
type SubmitQueue struct {
	sorted bool
	items  []SubmitRequest
}

// NewQueue Creates a new queue with the items present.
// The resulting queue has not yet been sorted.
func NewQueue(items []SubmitRequest) *SubmitQueue {
	return &SubmitQueue{
		sorted: false,
		items:  items,
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
	sort.Sort(byPriorityDescending(q.items))
	sort.Stable(byEmergencyFirst(q.items))
	q.sorted = true
}

// Dequeue Dequeues an item from a sorted queue. Queue must be resorted.
func (q *SubmitQueue) Dequeue() (SubmitRequest, error) {
	if len(q.items) == 0 {
		return nil, ErrQueueEmpty
	}
	if !q.sorted {
		return nil, ErrQueueUnsorted
	}
	item := q.items[0]
	q.items = q.items[1:]
	return item, nil
}

// Peek Peeks an item from a sorted queue. Queue must be resorted.
func (q *SubmitQueue) Peek() (SubmitRequest, error) {
	if len(q.items) < 1 {
		return nil, ErrQueueEmpty
	}
	if !q.sorted {
		return nil, ErrQueueUnsorted
	}
	item := q.items[0]
	return item, nil
}

type byPriorityDescending []SubmitRequest

func (p byPriorityDescending) Len() int {
	return len(p)
}
func (p byPriorityDescending) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p byPriorityDescending) Less(i, j int) bool {
	return p[i].Priority() > p[j].Priority()
}

type byEmergencyFirst []SubmitRequest

func (p byEmergencyFirst) Len() int {
	return len(p)
}
func (p byEmergencyFirst) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p byEmergencyFirst) Less(i, j int) bool {
	return p[i].IsEmergency() && !p[j].IsEmergency()
}
