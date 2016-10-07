package submitqueue

import "testing"

type testSubmitRequest struct {
	priority    Priority
	isEmergency bool
}

func (r testSubmitRequest) Priority() Priority {
	return r.priority
}
func (r testSubmitRequest) IsEmergency() bool {
	return r.isEmergency
}
func (r testSubmitRequest) Sha1() string {
	return "test"
}
func (r testSubmitRequest) GetProject() Project {
	return nil
}
func (r testSubmitRequest) GetRepo() Repo {
	return nil
}
func (r testSubmitRequest) GetPR() PullRequest {
	return nil
}

func testRequest(priority Priority, isEmergency bool) SubmitRequest {
	return testSubmitRequest{
		priority:    priority,
		isEmergency: isEmergency,
	}
}

func newQueue() *SubmitQueue {
	items := []SubmitRequest{}
	queue := NewQueue(items)
	return queue
}

func mustDequeue(q *SubmitQueue, t *testing.T) SubmitRequest {
	item, err := q.Dequeue()
	if err != nil {
		t.Fatalf(err.Error())
	}
	return item
}

func TestEnqueue_ShouldMarkUnsorted(t *testing.T) {
	queue := newQueue()
	queue.Sort()
	if !queue.sorted {
		t.Fatalf("intial queue should not be sorted")
	}

	queue.Enqueue(testRequest(PNormal, false))

	if queue.sorted {
		t.Errorf("queue should not be sorted, but was")
	}
}

func TestEnqueue_ShouldAddItem(t *testing.T) {
	queue := newQueue()
	queue.Enqueue(testRequest(PNormal, false))
	queue.Sort()

	if _, err := queue.Dequeue(); err != nil {
		t.Fatalf("expected enqueue to enqueue item, but dequeue threw error: %s", err)
	}
}

func TestDequeue_ShouldStillBeMarkedSorted(t *testing.T) {
	queue := newQueue()
	queue.Enqueue(testRequest(PNormal, false))
	queue.Sort()
	if !queue.sorted {
		t.Fatalf("queue was not sorted")
	}

	mustDequeue(queue, t)
	if !queue.sorted {
		t.Fatalf("queue should still be sorted, but wasn't")
	}
}

func TestDequeue_ShouldFailOnUnsortedQueue(t *testing.T) {
	queue := newQueue()
	queue.Enqueue(testRequest(PNormal, false))
	if queue.sorted {
		t.Fatalf("intial queue should not be sorted")
	}

	if _, err := queue.Dequeue(); err != ErrQueueUnsorted {
		t.Fatalf("should error on dequeue from unsorted queue")
	}
}

func TestDequeue_ShouldFailOnEmptyQueue(t *testing.T) {
	queue := newQueue()

	if _, err := queue.Dequeue(); err != ErrQueueEmpty {
		t.Fatalf("should error on dequeue from empty queue")
	}
}

func TestDequeue_ShouldRemoveItem(t *testing.T) {
	queue := newQueue()
	queue.Enqueue(testRequest(PNormal, false))
	queue.Enqueue(testRequest(P2, false))
	queue.Sort()

	item1 := mustDequeue(queue, t)
	item2 := mustDequeue(queue, t)
	if item1 == item2 {
		t.Fatalf("item was not removed from queue")
	}
}

func TestSort_ShouldMarkQueueSorted(t *testing.T) {
	queue := newQueue()

	if queue.sorted {
		t.Fatalf("initial should not be sorted")
	}

	queue.Sort()

	if !queue.sorted {
		t.Fatalf("sorted queue should be marked sorted")
	}
}

func TestSort_ShouldSortByPriority(t *testing.T) {
	queue := newQueue()
	queue.Enqueue(testRequest(P2, false))
	queue.Enqueue(testRequest(P1, false))
	queue.Sort()

	currentPriority := P1
	for {
		item, err := queue.Dequeue()
		if err != nil {
			break
		}
		if item.Priority() > currentPriority {
			t.Fatalf("incorrect priority ordering")
		}
		currentPriority = item.Priority()
	}
}

func TestSort_ShouldSortEmergencyFirst(t *testing.T) {
	queue := newQueue()
	emergencyRequest := testRequest(P2, true)
	p1Request := testRequest(P1, false)
	p2Request := testRequest(P2, false)
	queue.Enqueue(p1Request)
	queue.Enqueue(emergencyRequest)
	queue.Enqueue(p2Request)
	queue.Sort()

	item := mustDequeue(queue, t)
	if item != emergencyRequest {
		t.Fatalf("expected emergency request, got %v", item.Priority())
	}
}

func TestSort_ShouldSortEmergencyThenByPriority(t *testing.T) {
	queue := newQueue()
	emergencyRequest := testRequest(P2, true)
	p1Request := testRequest(P1, false)
	p2Request := testRequest(P2, false)
	queue.Enqueue(p1Request)
	queue.Enqueue(emergencyRequest)
	queue.Enqueue(p2Request)
	queue.Sort()

	item := mustDequeue(queue, t)
	if item != emergencyRequest {
		t.Fatalf("expected emergency request, got %v", item.Priority())
	}
	currentPriority := P1
	for {
		item, err := queue.Dequeue()
		if err != nil {
			break
		}
		if item.Priority() > currentPriority {
			t.Fatalf("incorrect priority ordering")
		}
		currentPriority = item.Priority()
	}
}
