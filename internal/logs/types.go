// Package logs provides log file parsing for AI CLI tools.
//
// This package defines common types and interfaces for parsing JSONL logs
// from Claude Code, Codex CLI, and Gemini CLI. Each provider has specific
// log formats but they all get normalized to the common LogEntry type.
package logs

import "time"

// LogEntry represents a single log entry from any provider.
// Fields may be empty/zero depending on what the provider logs.
type LogEntry struct {
	// Timestamp when this log entry was created
	Timestamp time.Time

	// Type of entry: "request", "response", "error", or provider-specific
	Type string

	// Model used for this request (e.g., "claude-3-opus", "gpt-4o")
	Model string

	// ConversationID groups related messages together
	ConversationID string

	// MessageID uniquely identifies this specific message
	MessageID string

	// Token counts
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheCreateTokens int64
	TotalTokens       int64

	// Raw contains provider-specific fields not in the common schema
	Raw map[string]any
}

// CalculateTotalTokens computes TotalTokens from component counts.
// This is useful when TotalTokens isn't directly provided in the log.
func (e *LogEntry) CalculateTotalTokens() int64 {
	return e.InputTokens + e.OutputTokens + e.CacheReadTokens + e.CacheCreateTokens
}

// TokenUsage aggregates token counts across multiple log entries.
type TokenUsage struct {
	InputTokens       int64
	OutputTokens      int64
	CacheReadTokens   int64
	CacheCreateTokens int64
	TotalTokens       int64

	// ByModel breaks down token usage by model
	ByModel map[string]*ModelTokenUsage
}

// NewTokenUsage creates an initialized TokenUsage.
func NewTokenUsage() *TokenUsage {
	return &TokenUsage{
		ByModel: make(map[string]*ModelTokenUsage),
	}
}

// Add accumulates tokens from a log entry.
func (u *TokenUsage) Add(entry *LogEntry) {
	u.InputTokens += entry.InputTokens
	u.OutputTokens += entry.OutputTokens
	u.CacheReadTokens += entry.CacheReadTokens
	u.CacheCreateTokens += entry.CacheCreateTokens
	u.TotalTokens += entry.InputTokens + entry.OutputTokens +
		entry.CacheReadTokens + entry.CacheCreateTokens

	if entry.Model != "" {
		if u.ByModel == nil {
			u.ByModel = make(map[string]*ModelTokenUsage)
		}
		mu, ok := u.ByModel[entry.Model]
		if !ok {
			mu = &ModelTokenUsage{Model: entry.Model}
			u.ByModel[entry.Model] = mu
		}
		mu.InputTokens += entry.InputTokens
		mu.OutputTokens += entry.OutputTokens
		mu.TotalTokens += entry.InputTokens + entry.OutputTokens +
			entry.CacheReadTokens + entry.CacheCreateTokens
	}
}

// ModelTokenUsage tracks token consumption for a specific model.
type ModelTokenUsage struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

// ScanResult contains parsed log data from a Scanner.
type ScanResult struct {
	// Provider identifies the source (e.g., "claude", "codex", "gemini")
	Provider string

	// TotalEntries is the number of log lines encountered
	TotalEntries int

	// ParsedEntries is the number successfully parsed
	ParsedEntries int

	// ParseErrors is the number that failed to parse
	ParseErrors int

	// Since is the start of the time window that was scanned
	Since time.Time

	// Until is the end of the time window (usually now)
	Until time.Time

	// Entries contains all successfully parsed log entries
	Entries []*LogEntry
}

// TokenUsage aggregates all entries into a single TokenUsage.
func (r *ScanResult) TokenUsage() *TokenUsage {
	usage := NewTokenUsage()
	for _, entry := range r.Entries {
		usage.Add(entry)
	}
	return usage
}

// FilterByModel returns entries matching the given model name.
func (r *ScanResult) FilterByModel(model string) []*LogEntry {
	var filtered []*LogEntry
	for _, entry := range r.Entries {
		if entry.Model == model {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// FilterByType returns entries matching the given type.
func (r *ScanResult) FilterByType(entryType string) []*LogEntry {
	var filtered []*LogEntry
	for _, entry := range r.Entries {
		if entry.Type == entryType {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

// Models returns a list of unique model names in the result.
func (r *ScanResult) Models() []string {
	seen := make(map[string]bool)
	var models []string
	for _, entry := range r.Entries {
		if entry.Model != "" && !seen[entry.Model] {
			seen[entry.Model] = true
			models = append(models, entry.Model)
		}
	}
	return models
}
