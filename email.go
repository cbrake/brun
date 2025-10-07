package brun

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"
)

// EmailConfig represents the configuration for an Email unit
type EmailConfig struct {
	UnitConfig    `yaml:",inline"`
	To            []string `yaml:"to"`
	From          string   `yaml:"from"`
	SubjectPrefix string   `yaml:"subject_prefix,omitempty"`
	SMTPHost      string   `yaml:"smtp_host"`
	SMTPPort      int      `yaml:"smtp_port,omitempty"`
	SMTPUser      string   `yaml:"smtp_user,omitempty"`
	SMTPPassword  string   `yaml:"smtp_password,omitempty"`
	SMTPUseTLS    *bool    `yaml:"smtp_use_tls,omitempty"`
	IncludeOutput *bool    `yaml:"include_output,omitempty"`
	LimitLines    int      `yaml:"limit_lines,omitempty"`
}

// EmailUnit sends email notifications
type EmailUnit struct {
	name           string
	to             []string
	from           string
	subjectPrefix  string
	smtpHost       string
	smtpPort       int
	smtpUser       string
	smtpPassword   string
	smtpUseTLS     bool
	includeOutput  bool
	limitLines     int
	output         string // Output from the triggering unit
	triggeringUnit string // Name of the unit that triggered this email
	triggerError   error  // Error from the triggering unit (if any)
	onSuccess      []string
	onFailure      []string
	always         []string
}

// NewEmailUnit creates a new Email unit
func NewEmailUnit(name string, to []string, from, subjectPrefix, smtpHost string, smtpPort int,
	smtpUser, smtpPassword string, smtpUseTLS, includeOutput bool, limitLines int,
	onSuccess, onFailure, always []string) *EmailUnit {
	return &EmailUnit{
		name:          name,
		to:            to,
		from:          from,
		subjectPrefix: subjectPrefix,
		smtpHost:      smtpHost,
		smtpPort:      smtpPort,
		smtpUser:      smtpUser,
		smtpPassword:  smtpPassword,
		smtpUseTLS:    smtpUseTLS,
		includeOutput: includeOutput,
		limitLines:    limitLines,
		onSuccess:     onSuccess,
		onFailure:     onFailure,
		always:        always,
	}
}

// Name returns the unit name
func (e *EmailUnit) Name() string {
	return e.name
}

// Type returns the unit type
func (e *EmailUnit) Type() string {
	return "email"
}

// SetOutput sets the output data from the triggering unit
func (e *EmailUnit) SetOutput(output string) {
	e.output = output
}

// SetTriggeringUnit sets the name of the unit that triggered this email
func (e *EmailUnit) SetTriggeringUnit(unitName string) {
	e.triggeringUnit = unitName
}

// SetTriggerError sets the error from the triggering unit
func (e *EmailUnit) SetTriggerError(err error) {
	e.triggerError = err
}

// Run executes the email unit
func (e *EmailUnit) Run(ctx context.Context) error {
	log.Printf("Running email unit '%s'", e.name)

	// Prepare email content
	timestamp := time.Now().Format(time.RFC3339)
	unitName := e.triggeringUnit
	if unitName == "" {
		unitName = "unknown"
	}

	// Build subject: <prefix>: <unit-name>:<success|fail>
	status := "success"
	if e.triggerError != nil {
		status = "fail"
	}

	subject := ""
	if e.subjectPrefix != "" {
		subject = e.subjectPrefix + ": "
	}
	subject += fmt.Sprintf("%s:%s", unitName, status)

	// Build body
	var body strings.Builder
	body.WriteString(fmt.Sprintf("Triggered by unit: %s\n", unitName))
	body.WriteString(fmt.Sprintf("Timestamp: %s\n\n", timestamp))

	if e.includeOutput && e.output != "" {
		body.WriteString("Output:\n")
		body.WriteString("-------\n")

		// Apply line limiting if configured
		output := e.output
		if e.limitLines > 0 {
			lines := strings.Split(output, "\n")
			if len(lines) > e.limitLines {
				// Keep last N lines
				lines = lines[len(lines)-e.limitLines:]
				output = strings.Join(lines, "\n")
				body.WriteString(fmt.Sprintf("(Showing last %d lines of %d total)\n", e.limitLines, len(strings.Split(e.output, "\n"))))
			}
		}

		body.WriteString(output)
		body.WriteString("\n")
	} else if !e.includeOutput {
		body.WriteString("(Output not included)\n")
	} else {
		body.WriteString("(No output captured)\n")
	}

	// Send email
	if err := e.sendEmail(subject, body.String()); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email unit '%s' completed, sent to %s", e.name, strings.Join(e.to, ", "))
	return nil
}

// sendEmail sends an email using SMTP
func (e *EmailUnit) sendEmail(subject, body string) error {
	// Build the email message
	message := e.buildMessage(subject, body)

	// Determine server address
	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	// Prepare authentication if credentials provided
	var auth smtp.Auth
	if e.smtpUser != "" && e.smtpPassword != "" {
		auth = smtp.PlainAuth("", e.smtpUser, e.smtpPassword, e.smtpHost)
	}

	// Send with or without TLS
	if e.smtpUseTLS {
		return e.sendEmailTLS(addr, auth, message)
	}

	// Send without TLS (plain SMTP)
	return smtp.SendMail(addr, auth, e.from, e.to, []byte(message))
}

// buildMessage constructs the RFC 5322 email message
func (e *EmailUnit) buildMessage(subject, body string) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("From: %s\r\n", e.from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.to, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	return msg.String()
}

// sendEmailTLS sends email with TLS encryption
func (e *EmailUnit) sendEmailTLS(addr string, auth smtp.Auth, message string) error {
	// Connect to the SMTP server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName:         e.smtpHost,
		InsecureSkipVerify: false,
	}

	if err = client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate if credentials provided
	if auth != nil {
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	// Set sender
	if err = client.Mail(e.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range e.to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// Quit
	return client.Quit()
}

// OnSuccess returns the list of units to trigger on success
func (e *EmailUnit) OnSuccess() []string {
	return e.onSuccess
}

// OnFailure returns the list of units to trigger on failure
func (e *EmailUnit) OnFailure() []string {
	return e.onFailure
}

// Always returns the list of units to always trigger
func (e *EmailUnit) Always() []string {
	return e.always
}
