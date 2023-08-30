package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dimfeld/httptreemux/v5"
)

type validator interface {
	Validate(body []byte) error
}

// Param returns the web call parameters from the request.
func Param(r *http.Request, key string) string {
	m := httptreemux.ContextParams(r.Context())
	return m[key]
}

// Decode reads the body of an HTTP request looking for a JSON document. The
// body is decoded into the provided value.
func Decode(r *http.Request, model any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read the body: %w", err)
	}

	if v, ok := model.(validator); ok {
		if err := v.Validate(body); err != nil {
			return fmt.Errorf("unable to validate the model: %w", err)
		}
	}

	if err := json.Unmarshal(body, model); err != nil {
		return fmt.Errorf("unable to unmarshal the data: %w", err)
	}

	return nil
}
