package output

import (
	"fmt"
	"os"
)

// Error outputs an error in the appropriate format
func (f *Formatter) Error(err error) error {
	if f.IsJSON() {
		return f.JSON(NewError(err.Error()))
	}
	return err
}

// ErrorMsg outputs an error message in the appropriate format
func (f *Formatter) ErrorMsg(msg string) error {
	if f.IsJSON() {
		return f.JSON(NewError(msg))
	}
	return fmt.Errorf("%s", msg)
}

// ErrorWithCode outputs an error with a code in the appropriate format
func (f *Formatter) ErrorWithCode(code, msg string) error {
	if f.IsJSON() {
		return f.JSON(NewErrorWithCode(code, msg))
	}
	return fmt.Errorf("[%s] %s", code, msg)
}

// PrintError writes an error to stderr and returns an error for JSON mode
func PrintError(err error, jsonMode bool) error {
	if jsonMode {
		return WriteJSON(os.Stdout, NewError(err.Error()), true)
	}
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	return err
}

// Fatal prints an error and exits
func Fatal(err error, jsonMode bool) {
	if jsonMode {
		WriteJSON(os.Stdout, NewError(err.Error()), true)
	} else {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	os.Exit(1)
}

// FatalMsg prints an error message and exits
func FatalMsg(msg string, jsonMode bool) {
	Fatal(fmt.Errorf("%s", msg), jsonMode)
}
