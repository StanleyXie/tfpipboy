package orchestrator

// ExecutorEvent represents events emitted by the executor
// These events are sent via channels to avoid locking and deadlocks
type ExecutorEvent struct {
	Type      ExecutorEventType
	JobID     string
	Timestamp int64
	Data      interface{}
}

// ExecutorEventType defines the type of executor event
type ExecutorEventType string

const (
	// EventTerraformOutput is emitted for each line of terraform output
	EventTerraformOutput ExecutorEventType = "terraform_output"

	// EventPlanResult is emitted when a plan operation completes with a summary
	EventPlanResult ExecutorEventType = "plan_result"

	// EventJobStatus is emitted when job status changes
	EventJobStatus ExecutorEventType = "job_status"
)

// TerraformOutputEvent contains terraform output line data
type TerraformOutputEvent struct {
	Component string // "terraform", "init", "plan", etc.
	Line      string // The output line
}

// PlanResultEvent contains plan summary data
type PlanResultEvent struct {
	Summary          string // Raw summary (e.g., "Plan: 20 to add, 0 to change, 0 to destroy.")
	FormattedSummary string // Formatted for display (e.g., "+20")
}

// JobStatusEvent contains job status change data
type JobStatusEvent struct {
	Status   JobStatus
	Progress string
	Error    string
}

// ExecutorEventBus handles event publishing and subscription
// This is the async communication layer between Executor and MessageRouter
type ExecutorEventBus struct {
	// Output channel for all events (buffered to prevent blocking)
	eventChan chan *ExecutorEvent

	// Closed when bus is shut down
	done chan struct{}
}

// NewExecutorEventBus creates a new event bus
func NewExecutorEventBus(bufferSize int) *ExecutorEventBus {
	if bufferSize <= 0 {
		bufferSize = 10000 // Large buffer to prevent blocking
	}

	return &ExecutorEventBus{
		eventChan: make(chan *ExecutorEvent, bufferSize),
		done:      make(chan struct{}),
	}
}

// Publish sends an event to the bus (non-blocking)
// Returns false if the bus is closed or buffer is full
func (eb *ExecutorEventBus) Publish(event *ExecutorEvent) bool {
	select {
	case eb.eventChan <- event:
		return true
	case <-eb.done:
		return false
	default:
		// Buffer full, drop event (non-blocking)
		// In production, you might want to log this
		return false
	}
}

// Subscribe returns the read-only event channel
func (eb *ExecutorEventBus) Subscribe() <-chan *ExecutorEvent {
	return eb.eventChan
}

// Close shuts down the event bus
func (eb *ExecutorEventBus) Close() {
	close(eb.done)
	close(eb.eventChan)
}
