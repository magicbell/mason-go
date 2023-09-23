package awslocal_test

import (
	"testing"

	"github.com/magicbell-io/mason-go/awslocal"
)

func Test_AwsLocalCfg(t *testing.T) {
	_, error := awslocal.NewAWSLocalCfg("localhost", "8000")

	if error != nil {
		t.Fatalf("expected no error, got %v", error)
	}

}
