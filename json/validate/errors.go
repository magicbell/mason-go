package validate

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// FieldError is used to indicate an error with a specific request field.
type FieldError struct {
	Message string `json:"message"`
}

// FieldErrors represents a collection of field errors.
type FieldErrors struct {
	Errors []FieldError `json:"errors"`
}

// Error implements the error interface on FieldErrors.
func (fe FieldErrors) Error() string {
	d, err := json.Marshal(fe)
	if err != nil {
		return err.Error()
	}
	return string(d)
}

func NewFieldErrors(msgs []string) FieldErrors {
	errs := make([]FieldError, 0, len(msgs))
	for _, msg := range msgs {
		errs = append(errs, FieldError{
			Message: msg,
		})
	}
	return FieldErrors{Errors: errs}
}

func toFieldErrors(result *gojsonschema.Result) FieldErrors {
	errs := make([]FieldError, 0, len(result.Errors()))
	for _, res := range result.Errors() {
		switch res.(type) {
		case *gojsonschema.NumberAllOfError, *gojsonschema.NumberAnyOfError, *gojsonschema.NumberOneOfError:
			continue
		default:
			errs = append(errs, FieldError{
				Message: newError(res),
			})
		}
	}
	return FieldErrors{Errors: errs}
}

// IsFieldErrors checks if an error of type FieldErrors exists.
func IsFieldErrors(err error) bool {
	var fe FieldErrors
	return errors.As(err, &fe)
}

// GetFieldErrors returns a copy of the FieldErrors pointer.
func GetFieldErrors(err error) FieldErrors {
	var fe FieldErrors
	if !errors.As(err, &fe) {
		return FieldErrors{}
	}
	return fe
}

// =============================================================================

func newError(resErr gojsonschema.ResultError) string {
	switch resErr.(type) {
	case *gojsonschema.RequiredError:
		return fmt.Sprintf("Param '%s' is missing", resErr.Details()["property"])
	case *gojsonschema.StringLengthLTEError:
		return fmt.Sprintf("Param '%s' is too long", resErr.Field())
	case *gojsonschema.ArrayMinItemsError:
		return fmt.Sprintf("Param '%s' must contain atleast %d items", resErr.Field(), resErr.Details()["min"])
	case *gojsonschema.ArrayMaxItemsError:
		return fmt.Sprintf("Param '%s' must contain at most %d items", resErr.Field(), resErr.Details()["max"])
	case *gojsonschema.AdditionalPropertyNotAllowedError:
		return fmt.Sprintf("Param '%s' doesn't allow key: %s", resErr.Field(), resErr.Details()["property"])
	case *gojsonschema.InvalidTypeError:
		return fmt.Sprintf("Param '%s' should be of type %s", resErr.Field(), resErr.Details()["expected"])
	case *gojsonschema.DoesNotMatchPatternError:
		return fmt.Sprintf("Param '%s' should match pattern %s", resErr.Field(), resErr.Details()["pattern"])
	case *gojsonschema.DoesNotMatchFormatError:
		return fmt.Sprintf("Param '%s' should be a valid %s", resErr.Field(), resErr.Details()["format"])

	// case *gojsonschema.FalseError:
	// case *gojsonschema.InvalidTypeError:
	// case *gojsonschema.NumberAnyOfError:
	// case *gojsonschema.NumberOneOfError:
	// case *gojsonschema.NumberAllOfError:
	// case *gojsonschema.NumberNotError:
	// case *gojsonschema.MissingDependencyError:
	// case *gojsonschema.InternalError:
	// case *gojsonschema.ConstError:
	// case *gojsonschema.EnumError:
	// case *gojsonschema.ArrayNoAdditionalItemsError:
	// case *gojsonschema.ArrayMaxItemsError:
	// case *gojsonschema.ItemsMustBeUniqueError:
	// case *gojsonschema.ArrayContainsError:
	// case *gojsonschema.ArrayMinPropertiesError:
	// case *gojsonschema.ArrayMaxPropertiesError:
	// case *gojsonschema.InvalidPropertyPatternError:
	// case *gojsonschema.InvalidPropertyNameError:
	// case *gojsonschema.StringLengthGTEError:
	// case *gojsonschema.MultipleOfError:
	// case *gojsonschema.NumberGTEError:
	// case *gojsonschema.NumberGTError:
	// case *gojsonschema.NumberLTEError:
	// case *gojsonschema.NumberLTError:
	// case *gojsonschema.ConditionThenError:
	// case *gojsonschema.ConditionElseError:

	default:
		return fmt.Sprintf("[%T]: %s", resErr, resErr.Description())
	}
}
