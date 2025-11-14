package orchestrator

import (
	"regexp"
	"strings"
	"time"
)

// LiveBoardEvent represents an event extracted from terraform output for liveboard updates
type LiveBoardEvent struct {
	InstanceID string
	EventType  string // "status", "step", "error", "plan_result", "progress"
	Timestamp  time.Time
	Data       map[string]interface{}
	Message    string // Human-readable message
}

// EventFilter extracts events from terraform output messages
type EventFilter struct {
	// Regex patterns for event detection
	planSummaryPattern *regexp.Regexp
	errorPattern       *regexp.Regexp
	refreshPattern     *regexp.Regexp
	resourcePattern    *regexp.Regexp
	noChangesPattern   *regexp.Regexp
}

// NewEventFilter creates a new event filter
func NewEventFilter() *EventFilter {
	return &EventFilter{
		// Plan summary: "Plan: 3 to add, 2 to change, 1 to destroy."
		planSummaryPattern: regexp.MustCompile(`Plan:\s+(\d+)\s+to\s+add,\s+(\d+)\s+to\s+change,\s+(\d+)\s+to\s+destroy`),

		// Error patterns
		errorPattern: regexp.MustCompile(`(?i)(error:|failed|fatal)`),

		// Refresh operations
		refreshPattern: regexp.MustCompile(`(?i)(refreshing state|reading\.\.\.)`),

		// Resource changes
		resourcePattern: regexp.MustCompile(`^\s*[+~-]\s+`),

		// No changes message
		noChangesPattern: regexp.MustCompile(`(?i)no changes.*infrastructure matches`),
	}
}

// ExtractEvents analyzes a message and extracts any events
func (ef *EventFilter) ExtractEvents(msg *Message) []*LiveBoardEvent {
	events := make([]*LiveBoardEvent, 0)

	cleanContent := StripANSI(msg.Content)

	// Check for plan summary
	if matches := ef.planSummaryPattern.FindStringSubmatch(cleanContent); len(matches) > 0 {
		add := matches[1]
		change := matches[2]
		destroy := matches[3]

		// Format plan result
		planResult := formatPlanSummaryForLiveBoard(cleanContent)

		event := &LiveBoardEvent{
			InstanceID: msg.InstanceID,
			EventType:  "plan_result",
			Timestamp:  msg.Timestamp,
			Data: map[string]interface{}{
				"add":     add,
				"change":  change,
				"destroy": destroy,
			},
			Message: planResult,
		}
		events = append(events, event)
	}

	// Check for "no changes" message
	if ef.noChangesPattern.MatchString(cleanContent) {
		event := &LiveBoardEvent{
			InstanceID: msg.InstanceID,
			EventType:  "plan_result",
			Timestamp:  msg.Timestamp,
			Data:       map[string]interface{}{},
			Message:    "No changes",
		}
		events = append(events, event)
	}

	// Check for errors
	if ef.errorPattern.MatchString(cleanContent) && msg.Level >= MessageLevelError {
		event := &LiveBoardEvent{
			InstanceID: msg.InstanceID,
			EventType:  "error",
			Timestamp:  msg.Timestamp,
			Data:       map[string]interface{}{},
			Message:    cleanContent,
		}
		events = append(events, event)
	}

	// Extract step/component changes from metadata
	if msg.Component != "" && msg.Component != "terraform" {
		// Component changed, likely a step change
		event := &LiveBoardEvent{
			InstanceID: msg.InstanceID,
			EventType:  "step",
			Timestamp:  msg.Timestamp,
			Data: map[string]interface{}{
				"step": msg.Component,
			},
			Message: msg.Component,
		}
		events = append(events, event)
	}

	// Extract status changes from metadata
	if status, ok := msg.Metadata["status"].(JobStatus); ok {
		event := &LiveBoardEvent{
			InstanceID: msg.InstanceID,
			EventType:  "status",
			Timestamp:  msg.Timestamp,
			Data: map[string]interface{}{
				"status": status,
			},
			Message: string(status),
		}
		events = append(events, event)
	}

	return events
}

// EventQueueProcessor processes messages and routes events to liveboard queue
type EventQueueProcessor struct {
	filter     *EventFilter
	eventQueue chan *LiveBoardEvent
	instanceID string
}

// NewEventQueueProcessor creates a processor that extracts events and sends to queue
func NewEventQueueProcessor(instanceID string, eventQueue chan *LiveBoardEvent) *EventQueueProcessor {
	return &EventQueueProcessor{
		filter:     NewEventFilter(),
		eventQueue: eventQueue,
		instanceID: instanceID,
	}
}

// ProcessMessage extracts events from message and sends to queue
func (eqp *EventQueueProcessor) ProcessMessage(msg *Message) error {
	// Extract events from message
	events := eqp.filter.ExtractEvents(msg)

	// Send events to queue (non-blocking)
	for _, event := range events {
		select {
		case eqp.eventQueue <- event:
			// Event queued successfully
		default:
			// Queue full, skip event (liveboard will be slightly stale)
		}
	}

	return nil
}

// Close closes the processor
func (eqp *EventQueueProcessor) Close() error {
	return nil
}

// LiveBoardBatchUpdater processes events from queue in batches and updates liveboard
type LiveBoardBatchUpdater struct {
	eventQueue chan *LiveBoardEvent
	liveBoard  *LiveBoard
	updateRate time.Duration
	stopChan   chan struct{}
	batchSize  int
}

// NewLiveBoardBatchUpdater creates a new batch updater
func NewLiveBoardBatchUpdater(eventQueue chan *LiveBoardEvent, liveBoard *LiveBoard) *LiveBoardBatchUpdater {
	return &LiveBoardBatchUpdater{
		eventQueue: eventQueue,
		liveBoard:  liveBoard,
		updateRate: 1 * time.Second, // Update every 1 second
		stopChan:   make(chan struct{}),
		batchSize:  100, // Process up to 100 events per batch
	}
}

// Start begins processing events from the queue
func (lbu *LiveBoardBatchUpdater) Start() {
	go lbu.processLoop()
}

// Stop stops the batch updater
func (lbu *LiveBoardBatchUpdater) Stop() {
	close(lbu.stopChan)
}

// processLoop processes events in batches at regular intervals
func (lbu *LiveBoardBatchUpdater) processLoop() {
	ticker := time.NewTicker(lbu.updateRate)
	defer ticker.Stop()

	for {
		select {
		case <-lbu.stopChan:
			// Drain remaining events before stopping
			lbu.drainAndUpdate()
			return

		case <-ticker.C:
			// Process batch of events
			lbu.drainAndUpdate()
		}
	}
}

// drainAndUpdate drains events from queue and updates liveboard
func (lbu *LiveBoardBatchUpdater) drainAndUpdate() {
	events := lbu.drainQueue()

	if len(events) == 0 {
		return
	}

	// Group events by instance
	instanceEvents := make(map[string][]*LiveBoardEvent)
	for _, event := range events {
		instanceEvents[event.InstanceID] = append(instanceEvents[event.InstanceID], event)
	}

	// Update liveboard for each instance
	for instanceID, events := range instanceEvents {
		// Process events in order for this instance
		for _, event := range events {
			lbu.applyEventToLiveBoard(instanceID, event)
		}
	}
}

// drainQueue drains up to batchSize events from the queue
func (lbu *LiveBoardBatchUpdater) drainQueue() []*LiveBoardEvent {
	events := make([]*LiveBoardEvent, 0, lbu.batchSize)

	for i := 0; i < lbu.batchSize; i++ {
		select {
		case event := <-lbu.eventQueue:
			events = append(events, event)
		default:
			// No more events in queue
			return events
		}
	}

	return events
}

// applyEventToLiveBoard applies a single event to the liveboard
func (lbu *LiveBoardBatchUpdater) applyEventToLiveBoard(instanceID string, event *LiveBoardEvent) {
	if lbu.liveBoard == nil {
		return
	}

	switch event.EventType {
	case "status":
		if status, ok := event.Data["status"].(JobStatus); ok {
			lbu.liveBoard.UpdateJobStatus(instanceID, status, event.Message)
		}

	case "step":
		if step, ok := event.Data["step"].(string); ok {
			lbu.liveBoard.UpdateJobStep(instanceID, step)
		}

	case "error":
		lbu.liveBoard.UpdateJobError(instanceID, event.Message)

	case "plan_result":
		lbu.liveBoard.UpdateJobPlanResult(instanceID, event.Message)

	case "progress":
		lbu.liveBoard.UpdateJobProgress(instanceID, event.Message)
	}
}

// formatPlanSummaryForLiveBoard formats a plan summary for display in liveboard
// This is moved from executor.go to be reusable
func formatPlanSummaryForLiveBoard(summary string) string {
	// Clean the summary
	clean := StripANSI(summary)
	clean = strings.TrimSpace(clean)

	// Extract numbers from "Plan: X to add, Y to change, Z to destroy"
	planPattern := regexp.MustCompile(`Plan:\s+(\d+)\s+to\s+add,\s+(\d+)\s+to\s+change,\s+(\d+)\s+to\s+destroy`)
	if matches := planPattern.FindStringSubmatch(clean); len(matches) == 4 {
		add := matches[1]
		change := matches[2]
		destroy := matches[3]

		// Format as compact string with symbols
		if add == "0" && change == "0" && destroy == "0" {
			return "No changes"
		}

		parts := []string{}
		if add != "0" {
			parts = append(parts, "+"+add)
		}
		if change != "0" {
			parts = append(parts, "~"+change)
		}
		if destroy != "0" {
			parts = append(parts, "-"+destroy)
		}

		return strings.Join(parts, " ")
	}

	// Check for "no changes" message
	if strings.Contains(strings.ToLower(clean), "no changes") {
		return "No changes"
	}

	// Return first line if can't parse
	lines := strings.Split(clean, "\n")
	if len(lines) > 0 {
		return lines[0]
	}

	return summary
}
