package orchestrator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MessageLevel represents the severity level of a message
type MessageLevel int

const (
	MessageLevelTrace MessageLevel = iota
	MessageLevelDebug
	MessageLevelInfo
	MessageLevelWarn
	MessageLevelError
)

func (ml MessageLevel) String() string {
	switch ml {
	case MessageLevelTrace:
		return "TRACE"
	case MessageLevelDebug:
		return "DEBUG"
	case MessageLevelInfo:
		return "INFO"
	case MessageLevelWarn:
		return "WARN"
	case MessageLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// MessageSource identifies where the message originated
type MessageSource string

const (
	MessageSourceOrchestrator MessageSource = "orchestrator"
	MessageSourceInstance     MessageSource = "instance"
	MessageSourceTerraform    MessageSource = "terraform"
	MessageSourceSystem       MessageSource = "system"
)

// Message represents a message in the pipeline
type Message struct {
	Timestamp  time.Time
	Level      MessageLevel
	Source     MessageSource
	InstanceID string // Empty for orchestrator messages
	Component  string // e.g., "init", "plan", "apply", "executor", "liveboard"
	Content    string // Original message content (may contain ANSI codes)
	Metadata   map[string]interface{}
}

// MessageFilter determines whether a message should be processed
type MessageFilter func(msg *Message) bool

// MessageProcessor processes messages (e.g., writes to log, updates UI)
type MessageProcessor interface {
	ProcessMessage(msg *Message) error
	Close() error
}

// MessagePipeline handles messages for a specific source (instance or orchestrator)
type MessagePipeline struct {
	mu         sync.RWMutex
	id         string // Instance ID or "orchestrator"
	processors []MessageProcessor
	filters    []MessageFilter
	msgChan    chan *Message
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewMessagePipeline creates a new message pipeline
func NewMessagePipeline(id string, bufferSize int) *MessagePipeline {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	return &MessagePipeline{
		id:         id,
		processors: make([]MessageProcessor, 0),
		filters:    make([]MessageFilter, 0),
		msgChan:    make(chan *Message, bufferSize),
		stopChan:   make(chan struct{}),
	}
}

// AddProcessor adds a message processor to the pipeline
func (mp *MessagePipeline) AddProcessor(processor MessageProcessor) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.processors = append(mp.processors, processor)
}

// AddFilter adds a message filter to the pipeline
func (mp *MessagePipeline) AddFilter(filter MessageFilter) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.filters = append(mp.filters, filter)
}

// Send sends a message to the pipeline
func (mp *MessagePipeline) Send(msg *Message) {
	select {
	case mp.msgChan <- msg:
	case <-mp.stopChan:
		// Pipeline stopped, discard message
	default:
		// Channel full, log warning and drop message
		fmt.Fprintf(os.Stderr, "WARNING: Message pipeline '%s' buffer full, dropping message\n", mp.id)
	}
}

// SendString is a convenience method to send a simple string message
func (mp *MessagePipeline) SendString(level MessageLevel, source MessageSource, component, content string) {
	msg := &Message{
		Timestamp:  time.Now(),
		Level:      level,
		Source:     source,
		InstanceID: mp.id,
		Component:  component,
		Content:    content,
		Metadata:   make(map[string]interface{}),
	}
	mp.Send(msg)
}

// Start begins processing messages
func (mp *MessagePipeline) Start() {
	mp.wg.Add(1)
	go mp.processLoop()
}

// Stop stops the pipeline and waits for processing to complete
func (mp *MessagePipeline) Stop() {
	close(mp.stopChan)
	mp.wg.Wait()

	// Close all processors
	mp.mu.Lock()
	defer mp.mu.Unlock()
	for _, processor := range mp.processors {
		processor.Close()
	}
}

// processLoop processes messages from the channel
func (mp *MessagePipeline) processLoop() {
	defer mp.wg.Done()

	for {
		select {
		case msg := <-mp.msgChan:
			mp.processMessage(msg)
		case <-mp.stopChan:
			// Drain remaining messages
			for {
				select {
				case msg := <-mp.msgChan:
					mp.processMessage(msg)
				default:
					return
				}
			}
		}
	}
}

// processMessage processes a single message through filters and processors
func (mp *MessagePipeline) processMessage(msg *Message) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	// Apply filters
	for _, filter := range mp.filters {
		if !filter(msg) {
			return // Message filtered out
		}
	}

	// Process message through all processors
	for _, processor := range mp.processors {
		if err := processor.ProcessMessage(msg); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to process message in pipeline '%s': %v\n", mp.id, err)
		}
	}
}

// FileLogProcessor writes messages to a log file
type FileLogProcessor struct {
	mu        sync.Mutex
	file      *os.File
	filePath  string
	stripANSI bool
}

// NewFileLogProcessor creates a new file log processor
func NewFileLogProcessor(filePath string, stripANSI bool) (*FileLogProcessor, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileLogProcessor{
		file:      file,
		filePath:  filePath,
		stripANSI: stripANSI,
	}, nil
}

// ProcessMessage writes the message to the log file
func (flp *FileLogProcessor) ProcessMessage(msg *Message) error {
	flp.mu.Lock()
	defer flp.mu.Unlock()

	content := msg.Content
	if flp.stripANSI {
		content = StripANSI(content)
	}

	// Format: [TIMESTAMP] [LEVEL] [SOURCE:COMPONENT] Message
	logLine := fmt.Sprintf("[%s] [%s] [%s:%s] %s\n",
		msg.Timestamp.Format("2006-01-02 15:04:05.000"),
		msg.Level.String(),
		msg.Source,
		msg.Component,
		content)

	_, err := flp.file.WriteString(logLine)
	return err
}

// Close closes the log file
func (flp *FileLogProcessor) Close() error {
	flp.mu.Lock()
	defer flp.mu.Unlock()
	if flp.file != nil {
		return flp.file.Close()
	}
	return nil
}

// ConsoleProcessor writes messages to console output
type ConsoleProcessor struct {
	mu        sync.Mutex
	writer    io.Writer
	stripANSI bool
}

// NewConsoleProcessor creates a new console processor
func NewConsoleProcessor(writer io.Writer, stripANSI bool) *ConsoleProcessor {
	if writer == nil {
		writer = os.Stdout
	}
	return &ConsoleProcessor{
		writer:    writer,
		stripANSI: stripANSI,
	}
}

// ProcessMessage writes the message to console
func (cp *ConsoleProcessor) ProcessMessage(msg *Message) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	content := msg.Content
	if cp.stripANSI {
		content = StripANSI(content)
	}

	// Write content directly (already formatted by source)
	_, err := fmt.Fprintln(cp.writer, content)
	return err
}

// Close closes the processor
func (cp *ConsoleProcessor) Close() error {
	return nil
}

// LiveBoardProcessor updates the liveboard display
type LiveBoardProcessor struct {
	mu        sync.Mutex
	liveBoard *LiveBoard
}

// NewLiveBoardProcessor creates a new liveboard processor
func NewLiveBoardProcessor(liveBoard *LiveBoard) *LiveBoardProcessor {
	return &LiveBoardProcessor{
		liveBoard: liveBoard,
	}
}

// ProcessMessage updates the liveboard based on message content
func (lbp *LiveBoardProcessor) ProcessMessage(msg *Message) error {
	lbp.mu.Lock()
	defer lbp.mu.Unlock()

	if lbp.liveBoard == nil {
		return nil
	}

	// Extract information from metadata
	if msg.InstanceID == "" {
		return nil // Orchestrator messages don't update job status
	}

	// Update liveboard based on component
	switch msg.Component {
	case "status":
		if status, ok := msg.Metadata["status"].(JobStatus); ok {
			lbp.liveBoard.UpdateJobStatus(msg.InstanceID, status, msg.Content)
		}
	case "step":
		if step, ok := msg.Metadata["step"].(string); ok {
			lbp.liveBoard.UpdateJobStep(msg.InstanceID, step)
		}
	case "error":
		lbp.liveBoard.UpdateJobError(msg.InstanceID, msg.Content)
	case "plan_result":
		lbp.liveBoard.UpdateJobPlanResult(msg.InstanceID, msg.Content)
	case "progress":
		lbp.liveBoard.UpdateJobProgress(msg.InstanceID, msg.Content)
	}

	return nil
}

// Close closes the processor
func (lbp *LiveBoardProcessor) Close() error {
	return nil
}

// Common message filters

// MinLevelFilter creates a filter that only passes messages at or above the specified level
func MinLevelFilter(minLevel MessageLevel) MessageFilter {
	return func(msg *Message) bool {
		return msg.Level >= minLevel
	}
}

// SourceFilter creates a filter that only passes messages from specified sources
func SourceFilter(sources ...MessageSource) MessageFilter {
	sourceMap := make(map[MessageSource]bool)
	for _, src := range sources {
		sourceMap[src] = true
	}
	return func(msg *Message) bool {
		return sourceMap[msg.Source]
	}
}

// ComponentFilter creates a filter that only passes messages from specified components
func ComponentFilter(components ...string) MessageFilter {
	componentMap := make(map[string]bool)
	for _, comp := range components {
		componentMap[comp] = true
	}
	return func(msg *Message) bool {
		return componentMap[msg.Component]
	}
}

// ExcludeComponentFilter creates a filter that excludes messages from specified components
func ExcludeComponentFilter(components ...string) MessageFilter {
	componentMap := make(map[string]bool)
	for _, comp := range components {
		componentMap[comp] = true
	}
	return func(msg *Message) bool {
		return !componentMap[msg.Component]
	}
}
