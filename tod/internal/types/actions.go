package types

import "time"

// CodeAction represents a discoverable user action from code analysis
type CodeAction struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Inputs      []UserInput       `json:"inputs,omitempty"`
	Expects     UserExpectation   `json:"expects,omitempty"`
	Implementation TechnicalDetails `json:"implementation,omitempty"`
	LastModified   time.Time        `json:"last_modified"`
}

// UserInput represents an input field for user actions
type UserInput struct {
	Name        string      `json:"name"`
	Label       string      `json:"label"`
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Example     string      `json:"example,omitempty"`
	Validation  string      `json:"validation,omitempty"`
	Description string      `json:"description,omitempty"`
}

// UserExpectation defines what should happen when an action is performed
type UserExpectation struct {
	Success   string   `json:"success"`
	Failure   string   `json:"failure"`
	Validates []string `json:"validates,omitempty"`
}

// TechnicalDetails contains implementation-specific information
type TechnicalDetails struct {
	Endpoint   string                 `json:"endpoint,omitempty"`
	Method     string                 `json:"method,omitempty"`
	SourceFile string                 `json:"source_file,omitempty"`
	Parameters map[string]Param       `json:"parameters,omitempty"`
	Responses  map[string]interface{} `json:"responses,omitempty"`
}

// Param represents a parameter with its metadata
type Param struct {
	Type        string      `json:"type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Validation  string      `json:"validation,omitempty"`
	Description string      `json:"description,omitempty"`
}