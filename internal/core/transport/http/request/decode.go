package core_http_request

import (
	"encoding/json"
	"fmt"
	"net/http"

	core_errors "github.com/dadqeds/todoapp/internal/core/errors"
	"github.com/go-playground/validator/v10"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

var requestValidator = validator.New()

type validatable interface {
	Validate() error
}

func DecodeAndValidateRequest(r *http.Request, dest any) error {
	body := http.MaxBytesReader(nil, r.Body, maxRequestBodyBytes)

	if err := json.NewDecoder(body).Decode(dest); err != nil {
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			core_errors.ErrInvalidArgument,
		)
	}

	var (
		err error
	)

	v, ok := dest.(validatable)
	if ok {
		err = v.Validate()
	} else {
		err = requestValidator.Struct(dest)
	}
	if err != nil {
		return fmt.Errorf(
			"request validator: %v: %w",
			err,
			core_errors.ErrInvalidArgument,
		)
	}

	return nil
}
