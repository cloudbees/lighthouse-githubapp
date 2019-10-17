package flags

import (
	"os"

	"github.com/sirupsen/logrus"
)

// BoolFlag a simple boolean flag
type BoolFlag struct {
	value    bool
	init     bool
	updated  bool
	envVar   string
	delegate *BoolFlag
}

// NewBoolFlag creates a new boolean flag
func NewBoolFlag(defaultValue bool, envVar string) *BoolFlag {
	return &BoolFlag{
		value:  defaultValue,
		envVar: envVar,
	}
}

// Value returns the value. If its been explicitly set it uses that value otherwise
// lets check if there's an environment variable. Otherwise lets use
func (f *BoolFlag) Value() bool {
	delegate := f.delegate
	if delegate != nil {
		return delegate.Value()
	}
	if f.updated || f.init {
		return f.value
	}
	if f.envVar != "" {
		text := os.Getenv(f.envVar)
		if text == "true" {
			f.value = true
		} else if text == "false" {
			f.value = false
		} else if text != "" {
			logrus.Warnf("environment variable %s has value %s for a boolean flag so is being ignored", f.envVar, text)
		}
	}
	f.init = true
	return f.value
}

// SetValue sets the value explicitly. Particularly useful in tests which then ignores env vars
func (f *BoolFlag) SetValue(value bool) {
	f.value = value
	f.updated = true
}

// With invokes the given function with this flag changed so that we can change flag
// for the duration of a test case and revert the value
func (f *BoolFlag) With(value bool, fn func() error) error {
	f.delegate = NewBoolFlag(value, f.envVar)
	defer f.clearDelegate()
	return fn()
}

func (f *BoolFlag) clearDelegate() {
	f.delegate = nil
}
