package awslocal_test

import (
	"testing"

	"github.com/magicbell-io/mason-go/awslocal"
)

func Test_AwsLocalCfg_New(t *testing.T) {
	_, error := awslocal.NewConfig("localhost", "8000")
	if error != nil {
		t.Fatalf("expected no error, got %v", error)
	}
}
