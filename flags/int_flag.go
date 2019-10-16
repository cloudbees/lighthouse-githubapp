package flags

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// IntFlag a simple int flag
type IntFlag struct {
	value    int
	init     bool
	updated  bool
	envVar   string
	delegate *IntFlag
}

// NewIntFlag creates a new boolean flag
func NewIntFlag(defaultValue int, envVar string) *IntFlag {
	return &IntFlag{
		value:  defaultValue,
		envVar: envVar,
	}
}

// Value returns the value. If its been explicitly set it uses that value otherwise
// lets check if there's an environment variable. Otherwise lets use
func (f *IntFlag) Value() int {
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
			var err error
			f.value, err = strconv.Atoi(text)
			if err != nil {
				logrus.Warnf("environment variable %s has value %s for an int flag which could not be parsed: %s", f.envVar,text, err.Error())
			}
		}
	}
	f.init = true
	return f.value
}

// SetValue sets the value explicitly. Particularly useful in tests which then ignores env vars
func (f *IntFlag) SetValue(value int) {
	f.value = value
	f.updated = true
}

// With invokes the given function with this flag changed so that we can change flag
// for the duration of a test case and revert the value
func (f *IntFlag) With(value int, fn func() error) error {
	f.delegate = NewIntFlag(value, f.envVar)
	defer f.clearDelegate()
	return fn()
}

func (f *IntFlag) clearDelegate() {
	f.delegate = nil
}
