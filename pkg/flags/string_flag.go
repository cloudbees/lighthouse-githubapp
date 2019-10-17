package flags

import (
	"os"
)

// StringFlag a simple boolean flag
type StringFlag struct {
	value    string
	updated  bool
	envVar   string
	init     bool
	delegate *StringFlag
}

// NewStringFlag creates a new string flag
func NewStringFlag(defaultValue string, envVar string) *StringFlag {
	return &StringFlag{
		value:  defaultValue,
		envVar: envVar,
	}
}

// Value returns the value. If its been explicitly set it uses that value otherwise
// lets check if there's an environment variable. Otherwise lets use
func (f *StringFlag) Value() string {
	delegate := f.delegate
	if delegate != nil {
		return delegate.Value()
	}
	if f.updated || f.init {
		return f.value
	}
	if f.envVar != "" {
		text := os.Getenv(f.envVar)
		if text != "" {
			f.value = text
		}
	}
	f.init = true
	return f.value
}

// SetValue sets the value explicitly. Particularly useful in tests which then ignores env vars
func (f *StringFlag) SetValue(value string) {
	f.value = value
	f.updated = true
}

// With invokes the given function with this flag changed so that we can change flag
// for the duration of a test case and revert the value
func (f *StringFlag) With(value string, fn func() error) error {
	f.delegate = NewStringFlag(value, f.envVar)
	defer f.clearDelegate()
	return fn()
}

func (f *StringFlag) clearDelegate() {
	f.delegate = nil
}
