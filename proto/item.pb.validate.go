// Code generated by protoc-gen-validate. DO NOT EDIT.
// source: item.proto

package proto

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"google.golang.org/protobuf/types/known/anypb"
)

// ensure the imports are used
var (
	_ = bytes.MinRead
	_ = errors.New("")
	_ = fmt.Print
	_ = utf8.UTFMax
	_ = (*regexp.Regexp)(nil)
	_ = (*strings.Reader)(nil)
	_ = net.IPv4len
	_ = time.Duration(0)
	_ = (*url.URL)(nil)
	_ = (*mail.Address)(nil)
	_ = anypb.Any{}
	_ = sort.Sort
)

// Validate checks the field values on Item with the rules defined in the proto
// definition for this message. If any rules are violated, the first error
// encountered is returned, or nil if there are no violations.
func (m *Item) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Item with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in ItemMultiError, or nil if none found.
func (m *Item) ValidateAll() error {
	return m.validate(true)
}

func (m *Item) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Id

	if all {
		switch v := interface{}(m.GetAttributes()).(type) {
		case interface{ ValidateAll() error }:
			if err := v.ValidateAll(); err != nil {
				errors = append(errors, ItemValidationError{
					field:  "Attributes",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		case interface{ Validate() error }:
			if err := v.Validate(); err != nil {
				errors = append(errors, ItemValidationError{
					field:  "Attributes",
					reason: "embedded message failed validation",
					cause:  err,
				})
			}
		}
	} else if v, ok := interface{}(m.GetAttributes()).(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return ItemValidationError{
				field:  "Attributes",
				reason: "embedded message failed validation",
				cause:  err,
			}
		}
	}

	// no validation rules for Score

	if len(errors) > 0 {
		return ItemMultiError(errors)
	}

	return nil
}

// ItemMultiError is an error wrapping multiple validation errors returned by
// Item.ValidateAll() if the designated constraints aren't met.
type ItemMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ItemMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ItemMultiError) AllErrors() []error { return m }

// ItemValidationError is the validation error returned by Item.Validate if the
// designated constraints aren't met.
type ItemValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ItemValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ItemValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ItemValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ItemValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ItemValidationError) ErrorName() string { return "ItemValidationError" }

// Error satisfies the builtin error interface
func (e ItemValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sItem.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ItemValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ItemValidationError{}

// Validate checks the field values on Items with the rules defined in the
// proto definition for this message. If any rules are violated, the first
// error encountered is returned, or nil if there are no violations.
func (m *Items) Validate() error {
	return m.validate(false)
}

// ValidateAll checks the field values on Items with the rules defined in the
// proto definition for this message. If any rules are violated, the result is
// a list of violation errors wrapped in ItemsMultiError, or nil if none found.
func (m *Items) ValidateAll() error {
	return m.validate(true)
}

func (m *Items) validate(all bool) error {
	if m == nil {
		return nil
	}

	var errors []error

	// no validation rules for Id

	for idx, item := range m.GetItems() {
		_, _ = idx, item

		if all {
			switch v := interface{}(item).(type) {
			case interface{ ValidateAll() error }:
				if err := v.ValidateAll(); err != nil {
					errors = append(errors, ItemsValidationError{
						field:  fmt.Sprintf("Items[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			case interface{ Validate() error }:
				if err := v.Validate(); err != nil {
					errors = append(errors, ItemsValidationError{
						field:  fmt.Sprintf("Items[%v]", idx),
						reason: "embedded message failed validation",
						cause:  err,
					})
				}
			}
		} else if v, ok := interface{}(item).(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return ItemsValidationError{
					field:  fmt.Sprintf("Items[%v]", idx),
					reason: "embedded message failed validation",
					cause:  err,
				}
			}
		}

	}

	if len(errors) > 0 {
		return ItemsMultiError(errors)
	}

	return nil
}

// ItemsMultiError is an error wrapping multiple validation errors returned by
// Items.ValidateAll() if the designated constraints aren't met.
type ItemsMultiError []error

// Error returns a concatenation of all the error messages it wraps.
func (m ItemsMultiError) Error() string {
	var msgs []string
	for _, err := range m {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// AllErrors returns a list of validation violation errors.
func (m ItemsMultiError) AllErrors() []error { return m }

// ItemsValidationError is the validation error returned by Items.Validate if
// the designated constraints aren't met.
type ItemsValidationError struct {
	field  string
	reason string
	cause  error
	key    bool
}

// Field function returns field value.
func (e ItemsValidationError) Field() string { return e.field }

// Reason function returns reason value.
func (e ItemsValidationError) Reason() string { return e.reason }

// Cause function returns cause value.
func (e ItemsValidationError) Cause() error { return e.cause }

// Key function returns key value.
func (e ItemsValidationError) Key() bool { return e.key }

// ErrorName returns error name.
func (e ItemsValidationError) ErrorName() string { return "ItemsValidationError" }

// Error satisfies the builtin error interface
func (e ItemsValidationError) Error() string {
	cause := ""
	if e.cause != nil {
		cause = fmt.Sprintf(" | caused by: %v", e.cause)
	}

	key := ""
	if e.key {
		key = "key for "
	}

	return fmt.Sprintf(
		"invalid %sItems.%s: %s%s",
		key,
		e.field,
		e.reason,
		cause)
}

var _ error = ItemsValidationError{}

var _ interface {
	Field() string
	Reason() string
	Key() bool
	Cause() error
	ErrorName() string
} = ItemsValidationError{}
