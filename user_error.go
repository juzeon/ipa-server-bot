package main

import (
	"errors"
	"fmt"
)

type UserError struct {
	Msg string
}

func (e *UserError) Error() string {
	return e.Msg
}

func userErr(msg string) error {
	return &UserError{Msg: msg}
}

func asUserError(v any) (string, bool) {
	switch e := v.(type) {
	case *UserError:
		return e.Msg, true
	case error:
		var ue *UserError
		if errors.As(e, &ue) {
			return ue.Msg, true
		}
	}
	return "", false
}

func handlerFailureMessage(rec any) (reply string, logDetail string) {
	if msg, ok := asUserError(rec); ok {
		return redactSecrets(msg), ""
	}
	return "Something went wrong. Please try again.", redactSecrets(fmt.Sprint(rec))
}
