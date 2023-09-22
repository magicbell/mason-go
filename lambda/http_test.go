package lambda_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/magicbell-io/gofoundation/lambda"
	"github.com/magicbell-io/gofoundation/web"
)

func TestAPIGWHandler(t *testing.T) {
	app := web.NewApp()
	app.Handle(http.MethodGet, "/test", "", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)

		return nil
	})

	lambda := lambda.HTTP{App: app}

	req := events.APIGatewayV2HTTPRequest{
		RequestContext: events.APIGatewayV2HTTPRequestContext{},
		RawPath:        "/test",
	}

	resp, err := lambda.APIGWHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// func TestALBHandler(t *testing.T) {
// 	app := web.NewApp()
// 	app.Router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 	})

// 	lambda := lambda.HTTP{App: app}

// 	req := events.ALBTargetGroupRequest{
// 		RequestContext: events.ALBTargetGroupRequestContext{},
// 		Path:           "/test",
// 	}

// 	resp, err := lambda.ALBHandler(context.Background(), req)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
// 	}
// }

// func TestAPIGWHandlerIntegration(t *testing.T) {
// 	app := web.NewApp()
// 	app.Router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
// 		w.WriteHeader(http.StatusOK)
// 	})

// 	lambda := lambda.HTTP{App: app}

// 	req := events.APIGatewayV2HTTPRequest{
// 		RequestContext: events.APIGatewayV2HTTPRequestContext{},
// 		RawPath:        "/test",
// 	}

// 	resp, err := lambda.APIGWHandler(context.Background(), req)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
// 	}

// 	// Test the response body
// 	rec := httptest.NewRecorder()
// 	rec.WriteHeader(http.StatusOK)
// 	app.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

// 	if rec.Body.String() != resp.Body {
// 		t.Errorf("expected response body %q, got %q", rec.Body.String(), resp.Body)
// 	}
// }

// func TestALBHandlerIntegration(t *testing.T) {
// 	app := web.NewApp()
// 	app.Handle(http.MethodGet, "/test", "", func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
// 		w.WriteHeader(http.StatusOK)
// 	})

// 	lambda := lambda.HTTP{App: app}

// 	req := events.ALBTargetGroupRequest{
// 		RequestContext: events.ALBTargetGroupRequestContext{},
// 		Path:           "/test",
// 	}

// 	resp, err := lambda.ALBHandler(context.Background(), req)
// 	if err != nil {
// 		t.Fatalf("unexpected error: %v", err)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		t.Errorf("expected status code %d, got %d", http.StatusOK, resp.StatusCode)
// 	}

// 	// Test the response body
// 	rec := httptest.NewRecorder()
// 	rec.WriteHeader(http.StatusOK)
// 	app.Router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/test", nil))

// 	if rec.Body.String() != resp.Body {
// 		t.Errorf("expected response body %q, got %q", rec.Body.String(), resp.Body)
// 	}
// }

// // END: c8f7e9d8jw3p
