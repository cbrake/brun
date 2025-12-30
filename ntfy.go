package brun

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// NtfyConfig represents the configuration for an Ntfy unit
type NtfyConfig struct {
	UnitConfig    `yaml:",inline"`
	Topic         string `yaml:"topic"`
	Server        string `yaml:"server,omitempty"`
	TitlePrefix   string `yaml:"title_prefix,omitempty"`
	Priority      string `yaml:"priority,omitempty"`
	Tags          string `yaml:"tags,omitempty"`
	IncludeOutput *bool  `yaml:"include_output,omitempty"`
	LimitLines    int    `yaml:"limit_lines,omitempty"`
}

// NtfyUnit sends notifications via ntfy.sh
type NtfyUnit struct {
	name           string
	topic          string
	server         string
	titlePrefix    string
	priority       string
	tags           string
	includeOutput  bool
	limitLines     int
	output         string
	triggeringUnit string
	triggerError   error
	onSuccess      []string
	onFailure      []string
	always         []string
}

// NewNtfyUnit creates a new Ntfy unit
func NewNtfyUnit(name, topic, server, titlePrefix, priority, tags string,
	includeOutput bool, limitLines int,
	onSuccess, onFailure, always []string) *NtfyUnit {
	return &NtfyUnit{
		name:          name,
		topic:         topic,
		server:        server,
		titlePrefix:   titlePrefix,
		priority:      priority,
		tags:          tags,
		includeOutput: includeOutput,
		limitLines:    limitLines,
		onSuccess:     onSuccess,
		onFailure:     onFailure,
		always:        always,
	}
}

// Name returns the unit name
func (n *NtfyUnit) Name() string {
	return n.name
}

// Type returns the unit type
func (n *NtfyUnit) Type() string {
	return "ntfy"
}

// SetOutput sets the output data from the triggering unit
func (n *NtfyUnit) SetOutput(output string) {
	n.output = output
}

// SetTriggeringUnit sets the name of the unit that triggered this notification
func (n *NtfyUnit) SetTriggeringUnit(unitName string) {
	n.triggeringUnit = unitName
}

// SetTriggerError sets the error from the triggering unit
func (n *NtfyUnit) SetTriggerError(err error) {
	n.triggerError = err
}

// Run executes the ntfy unit
func (n *NtfyUnit) Run(ctx context.Context) error {
	log.Printf("Running ntfy unit '%s'", n.name)

	// Build notification body
	body := n.buildBody()

	// Build title: <prefix>: <unit-name>:<success|fail>
	unitName := n.triggeringUnit
	if unitName == "" {
		unitName = "unknown"
	}
	status := "success"
	if n.triggerError != nil {
		status = "fail"
	}

	title := ""
	if n.titlePrefix != "" {
		title = n.titlePrefix + ": "
	}
	title += fmt.Sprintf("%s:%s", unitName, status)

	// Send notification
	if err := n.sendNotification(ctx, title, body); err != nil {
		return fmt.Errorf("failed to send ntfy notification: %w", err)
	}

	log.Printf("Ntfy unit '%s' completed, sent to %s/%s", n.name, n.server, n.topic)
	return nil
}

// buildBody constructs the notification body
func (n *NtfyUnit) buildBody() string {
	var body strings.Builder

	timestamp := time.Now().Format(time.RFC3339)
	unitName := n.triggeringUnit
	if unitName == "" {
		unitName = "unknown"
	}

	body.WriteString(fmt.Sprintf("Triggered by: %s\n", unitName))
	body.WriteString(fmt.Sprintf("Timestamp: %s\n", timestamp))

	if n.triggerError != nil {
		body.WriteString(fmt.Sprintf("Error: %v\n", n.triggerError))
	}

	if n.includeOutput && n.output != "" {
		body.WriteString("\nOutput:\n")

		output := n.output
		if n.limitLines > 0 {
			lines := strings.Split(output, "\n")
			if len(lines) > n.limitLines {
				lines = lines[len(lines)-n.limitLines:]
				output = strings.Join(lines, "\n")
				body.WriteString(fmt.Sprintf("(last %d of %d lines)\n", n.limitLines, len(strings.Split(n.output, "\n"))))
			}
		}

		body.WriteString(output)
	} else if !n.includeOutput {
		body.WriteString("\n(Output not included)")
	} else {
		body.WriteString("\n(No output captured)")
	}

	return body.String()
}

// sendNotification sends the notification to the ntfy server
func (n *NtfyUnit) sendNotification(ctx context.Context, title, body string) error {
	url := fmt.Sprintf("%s/%s", n.server, n.topic)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if title != "" {
		req.Header.Set("Title", title)
	}
	if n.priority != "" {
		req.Header.Set("Priority", n.priority)
	}
	if n.tags != "" {
		req.Header.Set("Tags", n.tags)
	}

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ntfy server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// OnSuccess returns the list of units to trigger on success
func (n *NtfyUnit) OnSuccess() []string {
	return n.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (n *NtfyUnit) OnFailure() []string {
	return n.onFailure
}

// Always returns the list of units to always trigger
func (n *NtfyUnit) Always() []string {
	return n.always
}
