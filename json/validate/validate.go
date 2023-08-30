// Package validate contains the support for validating models.
package validate

import (
	"errors"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// ErrBodyEmpty occurs when the body of the reponse was empty.
var ErrBodyEmpty = errors.New("body empty")

// Check validates the provided model against it's declared tags.
func Check(schemaDoc []byte, body []byte) error {
	if len(body) == 0 {
		return ErrBodyEmpty
	}

	result, err := gojsonschema.Validate(gojsonschema.NewBytesLoader(schemaDoc), gojsonschema.NewBytesLoader(body))
	if err != nil {
		return fmt.Errorf("json schema validate: %w", err)
	}

	if !result.Valid() {
		return toFieldErrors(result)
	}

	return nil
}
