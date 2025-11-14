package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// OutputMode determines what gets displayed on console
type OutputMode string

const (
	OutputModeLiveBoardOnly    OutputMode = "liveboard_only"    // Only liveboard during execution
	OutputModeLiveBoardDetails OutputMode = "liveboard_details" // Liveboard + instance details after completion
	OutputModeAllMessages      OutputMode = "all_messages"      // All messages from all pipelines
	OutputModeQuiet            OutputMode = "quiet"             // No console output except errors
)

// MessageRouterConfig configures the message router
type MessageRouterConfig struct {
	BaseLogDir       string
	ConsoleMode      OutputMode
	VerboseMode      bool // Enable DEBUG level messages
	TraceMode        bool // Enable TRACE level messages (most verbose)
	StripANSIInLogs  bool // Remove ANSI codes from log files
	StripANSIConsole bool // Remove ANSI codes from console output
	BufferSize       int  // Message channel buffer size per pipeline
}

// MessageRouter manages all message pipelines and routing
type MessageRouter struct {
	mu                   sync.RWMutex
	config               *MessageRouterConfig
	orchestratorPipeline *MessagePipeline
	instancePipelines    map[string]*MessagePipeline
	instanceOperations   map[string]TerraformOperation // Track operation for each instance
	instancePlanResults  map[string]string             // Track plan results for each instance
	liveBoard            *LiveBoard
	eventQueue           chan *LiveBoardEvent // FIFO queue for liveboard events
	batchUpdater         *LiveBoardBatchUpdater
	consoleBuffer        []*Message // Buffer messages during liveboard execution
	liveboardActive      bool
}

// NewMessageRouter creates a new message router
func NewMessageRouter(config *MessageRouterConfig) *MessageRouter {
	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	return &MessageRouter{
		config:              config,
		instancePipelines:   make(map[string]*MessagePipeline),
		instanceOperations:  make(map[string]TerraformOperation),
		instancePlanResults: make(map[string]string),
		eventQueue:          make(chan *LiveBoardEvent, 1000), // Event queue for batch updates
		consoleBuffer:       make([]*Message, 0),
	}
}

// InitializeOrchestrator initializes the orchestrator message pipeline
func (mr *MessageRouter) InitializeOrchestrator() error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Create orchestrator pipeline
	mr.orchestratorPipeline = NewMessagePipeline("orchestrator", mr.config.BufferSize)

	// Add log file processor
	logPath := filepath.Join(mr.config.BaseLogDir, "orchestrator.log")
	logProcessor, err := NewFileLogProcessor(logPath, mr.config.StripANSIInLogs)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator log processor: %w", err)
	}
	mr.orchestratorPipeline.AddProcessor(logProcessor)

	// Add console processor based on mode
	if mr.config.ConsoleMode == OutputModeAllMessages {
		consoleProcessor := NewConsoleProcessor(os.Stdout, mr.config.StripANSIConsole)
		mr.orchestratorPipeline.AddProcessor(consoleProcessor)
	}

	// Add level filter based on verbose/trace mode
	if mr.config.TraceMode {
		mr.orchestratorPipeline.AddFilter(MinLevelFilter(MessageLevelTrace))
	} else if mr.config.VerboseMode {
		mr.orchestratorPipeline.AddFilter(MinLevelFilter(MessageLevelDebug))
	} else {
		mr.orchestratorPipeline.AddFilter(MinLevelFilter(MessageLevelInfo))
	}

	// Start pipeline
	mr.orchestratorPipeline.Start()

	return nil
}

// InitializeInstance creates and initializes a message pipeline for an instance
// Implements dual-path architecture:
//
//	Path 1: ALL messages → FileLogProcessor → instance log file (with ANSI colors)
//	Path 2: ALL messages → EventQueueProcessor → Event Queue → Batch Updater → LiveBoard
//
// SetInstancePlanResult sets the plan result for an instance
func (mr *MessageRouter) SetInstancePlanResult(instanceID, planResult string) {
	mr.mu.Lock()
	defer mr.mu.Unlock()
	mr.instancePlanResults[instanceID] = planResult
}

// StartEventSubscriber starts a goroutine that subscribes to executor events
// This eliminates the need for the executor to hold locks while calling MessageRouter
func (mr *MessageRouter) StartEventSubscriber(eventBus *ExecutorEventBus) {
	go func() {
		for event := range eventBus.Subscribe() {
			switch event.Type {
			case EventPlanResult:
				// Handle plan result event
				if data, ok := event.Data.(*PlanResultEvent); ok {
					mr.mu.Lock()
					mr.instancePlanResults[event.JobID] = data.Summary
					mr.mu.Unlock()
				}

			case EventTerraformOutput:
				// Handle terraform output event
				if data, ok := event.Data.(*TerraformOutputEvent); ok {
					// Create message
					msg := &Message{
						InstanceID: event.JobID,
						Component:  data.Component,
						Level:      MessageLevelInfo,
						Content:    data.Line,
						Timestamp:  time.Unix(event.Timestamp, 0),
					}

					// Buffer the message if liveboard is active (for filtered output later)
					mr.BufferMessage(msg)

					// Also send to pipeline for logging
					mr.mu.RLock()
					pipeline := mr.instancePipelines[event.JobID]
					mr.mu.RUnlock()

					if pipeline != nil {
						pipeline.Send(msg)
					}
				}
			}
		}
	}()
}

func (mr *MessageRouter) InitializeInstance(instanceID string, operation TerraformOperation) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Store operation for this instance (used for filtering output)
	mr.instanceOperations[instanceID] = operation

	// Create instance pipeline
	pipeline := NewMessagePipeline(instanceID, mr.config.BufferSize)

	// ============================================================================
	// PATH 1: Log Writer - ALL messages go to log file
	// ============================================================================
	logPath := filepath.Join(mr.config.BaseLogDir, "instances", fmt.Sprintf("%s.log", instanceID))
	logProcessor, err := NewFileLogProcessor(logPath, false) // Keep ANSI codes in instance logs
	if err != nil {
		return fmt.Errorf("failed to create instance log processor: %w", err)
	}
	pipeline.AddProcessor(logProcessor)

	// ============================================================================
	// PATH 2: Event Filter → Queue → Batch Updater → LiveBoard
	// ============================================================================
	// Add event queue processor to extract events from messages
	eventQueueProcessor := NewEventQueueProcessor(instanceID, mr.eventQueue)
	pipeline.AddProcessor(eventQueueProcessor)

	// ============================================================================
	// Buffer Processor - Add BEFORE checking liveboardActive
	// In liveboard_details mode, we need to buffer messages even when LiveBoard is active
	// ============================================================================
	if mr.config.ConsoleMode == OutputModeLiveBoardDetails {
		bufferProcessor := NewBufferProcessor(mr)
		pipeline.AddProcessor(bufferProcessor)
	}

	// ============================================================================
	// Console Output - ONLY if NOT in LiveBoard mode (console snippet exclusivity)
	// ============================================================================
	if !mr.liveboardActive {
		// LiveBoard not active, allow console output based on mode
		switch mr.config.ConsoleMode {
		case OutputModeAllMessages:
			// All messages go to console immediately
			consoleProcessor := NewConsoleProcessor(os.Stdout, mr.config.StripANSIConsole)
			pipeline.AddProcessor(consoleProcessor)
		}
	}
	// Note: When liveboardActive=true, NO console processors are added (except buffer)
	// This enforces console snippet exclusivity during execution

	// Add level filter (applies to all processors)
	if mr.config.TraceMode {
		pipeline.AddFilter(MinLevelFilter(MessageLevelTrace))
	} else if mr.config.VerboseMode {
		pipeline.AddFilter(MinLevelFilter(MessageLevelDebug))
	} else {
		pipeline.AddFilter(MinLevelFilter(MessageLevelInfo))
	}

	// Start pipeline
	pipeline.Start()

	mr.instancePipelines[instanceID] = pipeline

	return nil
}

// GetInstancePipeline returns the message pipeline for an instance
func (mr *MessageRouter) GetInstancePipeline(instanceID string) *MessagePipeline {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return mr.instancePipelines[instanceID]
}

// GetOrchestratorPipeline returns the orchestrator message pipeline
func (mr *MessageRouter) GetOrchestratorPipeline() *MessagePipeline {
	mr.mu.RLock()
	defer mr.mu.RUnlock()
	return mr.orchestratorPipeline
}

// SetLiveBoard sets the liveboard for the router and starts the batch updater
func (mr *MessageRouter) SetLiveBoard(liveBoard *LiveBoard) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.liveBoard = liveBoard
	mr.liveboardActive = true

	// Create and start batch updater for event queue → liveboard updates
	mr.batchUpdater = NewLiveBoardBatchUpdater(mr.eventQueue, liveBoard)
	mr.batchUpdater.Start()
}

// StartLiveBoard marks liveboard as active and starts batch updater
func (mr *MessageRouter) StartLiveBoard() {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	mr.liveboardActive = true
	mr.consoleBuffer = make([]*Message, 0) // Reset buffer

	// Start batch updater if liveboard is set
	if mr.liveBoard != nil && mr.batchUpdater == nil {
		mr.batchUpdater = NewLiveBoardBatchUpdater(mr.eventQueue, mr.liveBoard)
		mr.batchUpdater.Start()
	}
}

// StopLiveBoard marks liveboard as stopped and flushes buffered messages
func (mr *MessageRouter) StopLiveBoard() {
	// Stop batch updater first
	mr.mu.Lock()
	if mr.batchUpdater != nil {
		mr.batchUpdater.Stop()
		mr.batchUpdater = nil
	}
	mr.mu.Unlock()

	// Wait a moment for any final terraform output to be buffered
	// (plan summary often comes at the very end)
	time.Sleep(100 * time.Millisecond)

	// Now stop buffering and capture the buffer
	mr.mu.Lock()
	mr.liveboardActive = false
	bufferLen := len(mr.consoleBuffer)
	consoleMode := mr.config.ConsoleMode
	mr.mu.Unlock()

	// Flush buffered messages if in LiveBoardDetails mode (outside lock to avoid deadlock)
	if consoleMode == OutputModeLiveBoardDetails && bufferLen > 0 {

		// Use Stdout for the filtered output section
		out := os.Stdout
		fmt.Fprintf(out, "\n"+colorBlue+"════════════════════════════════════════════════════════════════════════════════"+colorReset+"\n")
		fmt.Fprintf(out, colorBlue+"                        INSTANCE EXECUTION DETAILS                               "+colorReset+"\n")
		fmt.Fprintf(out, colorBlue+"════════════════════════════════════════════════════════════════════════════════"+colorReset+"\n\n")

		// Group messages by instance
		instanceMessages := make(map[string][]*Message)
		for _, msg := range mr.consoleBuffer {
			if msg.InstanceID != "" {
				instanceMessages[msg.InstanceID] = append(instanceMessages[msg.InstanceID], msg)
			}
		}

		// Process and print filtered messages for each instance
		for instanceID, messages := range instanceMessages {
			// Get operation and plan result for this instance
			operation := mr.instanceOperations[instanceID]
			planResult := mr.instancePlanResults[instanceID]

			// Create filter for this instance
			filter := NewTerraformOutputFilter(instanceID, operation)

			// Process all messages through the filter
			for _, msg := range messages {
				content := msg.Content
				if mr.config.StripANSIConsole {
					content = StripANSI(content)
				}
				filter.ProcessLine(content)
			}

			// If plan summary wasn't captured from messages, inject it from plan result
			if planResult != "" {
				if filter.planChanges == nil {
					filter.planChanges = &PlanChanges{Details: make([]string, 0)}
				}
				if filter.planChanges.Summary == "" {
					filter.planChanges.Summary = planResult
				}
			}

			// If operation is plan and no resource changes captured, try reading from plan file
			if operation == OpPlan && (filter.planChanges == nil || len(filter.planChanges.Details) == 0) {
				planFilePath := filepath.Join(mr.config.BaseLogDir, "..", "workspaces", instanceID, "terraform.tfplan.txt")
				if planFileData, err := os.ReadFile(planFilePath); err == nil {
					// Process plan file line by line to extract resource changes
					lines := strings.Split(string(planFileData), "\n")
					for _, line := range lines {
						filter.ProcessLine(line)
					}
				}
			}

			// Print filtered output to stdout
			fmt.Fprintf(out, "%s", filter.FormatOutput())
		}

		mr.consoleBuffer = nil // Clear buffer
	}
}

// BufferMessage adds a message to the console buffer
func (mr *MessageRouter) BufferMessage(msg *Message) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	// Buffer messages if in liveboard_details mode (regardless of liveboardActive flag)
	// This ensures we capture all terraform output for filtered display after execution
	if mr.config.ConsoleMode == OutputModeLiveBoardDetails {
		mr.consoleBuffer = append(mr.consoleBuffer, msg)
	}
}

// Shutdown stops all pipelines and closes processors
func (mr *MessageRouter) Shutdown() {
	mr.mu.Lock()

	// Stop batch updater first
	if mr.batchUpdater != nil {
		mr.batchUpdater.Stop()
		mr.batchUpdater = nil
	}

	// Stop orchestrator pipeline
	if mr.orchestratorPipeline != nil {
		mr.orchestratorPipeline.Stop()
	}

	// Stop all instance pipelines
	for _, pipeline := range mr.instancePipelines {
		pipeline.Stop()
	}

	mr.mu.Unlock()

	// Close event queue (after all pipelines stopped)
	close(mr.eventQueue)
}

// BufferProcessor buffers messages for later display
type BufferProcessor struct {
	router *MessageRouter
}

// NewBufferProcessor creates a new buffer processor
func NewBufferProcessor(router *MessageRouter) *BufferProcessor {
	return &BufferProcessor{
		router: router,
	}
}

// ProcessMessage buffers the message
func (bp *BufferProcessor) ProcessMessage(msg *Message) error {
	// Only buffer terraform output and errors
	if msg.Source == MessageSourceTerraform || msg.Level == MessageLevelError {
		bp.router.BufferMessage(msg)
	}
	return nil
}

// Close closes the processor
func (bp *BufferProcessor) Close() error {
	return nil
}

// Helper functions for sending common message types

// SendOrchestratorInfo sends an info message to the orchestrator pipeline
func (mr *MessageRouter) SendOrchestratorInfo(component, content string) {
	if mr.orchestratorPipeline != nil {
		mr.orchestratorPipeline.SendString(MessageLevelInfo, MessageSourceOrchestrator, component, content)
	}
}

// SendOrchestratorDebug sends a debug message to the orchestrator pipeline
func (mr *MessageRouter) SendOrchestratorDebug(component, content string) {
	if mr.orchestratorPipeline != nil {
		mr.orchestratorPipeline.SendString(MessageLevelDebug, MessageSourceOrchestrator, component, content)
	}
}

// SendOrchestratorError sends an error message to the orchestrator pipeline
func (mr *MessageRouter) SendOrchestratorError(component, content string) {
	if mr.orchestratorPipeline != nil {
		mr.orchestratorPipeline.SendString(MessageLevelError, MessageSourceOrchestrator, component, content)
	}
}

// SendInstanceStatus sends a status update for an instance
func (mr *MessageRouter) SendInstanceStatus(instanceID string, status JobStatus, message string) {
	pipeline := mr.GetInstancePipeline(instanceID)
	if pipeline != nil {
		msg := &Message{
			Timestamp:  time.Now(),
			Level:      MessageLevelInfo,
			Source:     MessageSourceInstance,
			InstanceID: instanceID,
			Component:  "status",
			Content:    message,
			Metadata: map[string]interface{}{
				"status": status,
			},
		}
		pipeline.Send(msg)
	}
}

// SendInstanceTerraformOutput sends terraform output for an instance
func (mr *MessageRouter) SendInstanceTerraformOutput(instanceID, component, content string) {
	pipeline := mr.GetInstancePipeline(instanceID)
	if pipeline != nil {
		msg := &Message{
			Timestamp:  time.Now(),
			Level:      MessageLevelInfo,
			Source:     MessageSourceTerraform,
			InstanceID: instanceID,
			Component:  component,
			Content:    content,
			Metadata:   make(map[string]interface{}),
		}
		pipeline.Send(msg)
	}
}

// SendInstanceError sends an error message for an instance
func (mr *MessageRouter) SendInstanceError(instanceID, component, content string) {
	pipeline := mr.GetInstancePipeline(instanceID)
	if pipeline != nil {
		msg := &Message{
			Timestamp:  time.Now(),
			Level:      MessageLevelError,
			Source:     MessageSourceInstance,
			InstanceID: instanceID,
			Component:  component,
			Content:    content,
			Metadata:   make(map[string]interface{}),
		}
		pipeline.Send(msg)
	}
}

// SendInstanceStep sends a step update for an instance
func (mr *MessageRouter) SendInstanceStep(instanceID, step string) {
	pipeline := mr.GetInstancePipeline(instanceID)
	if pipeline != nil {
		msg := &Message{
			Timestamp:  time.Now(),
			Level:      MessageLevelInfo,
			Source:     MessageSourceInstance,
			InstanceID: instanceID,
			Component:  "step",
			Content:    fmt.Sprintf("Starting step: %s", step),
			Metadata: map[string]interface{}{
				"step": step,
			},
		}
		pipeline.Send(msg)
	}
}

// SendInstancePlanResult sends a plan result for an instance
func (mr *MessageRouter) SendInstancePlanResult(instanceID, planResult string) {
	pipeline := mr.GetInstancePipeline(instanceID)
	if pipeline != nil {
		msg := &Message{
			Timestamp:  time.Now(),
			Level:      MessageLevelInfo,
			Source:     MessageSourceInstance,
			InstanceID: instanceID,
			Component:  "plan_result",
			Content:    planResult,
			Metadata:   make(map[string]interface{}),
		}
		pipeline.Send(msg)
	}
}
