// Package lambda provides support for running the app inside a Lambda function.
package lambda

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	lambdaproxy "github.com/awslabs/aws-lambda-go-api-proxy/core"
	"github.com/magicbell-io/gofoundation/web"
)

// HTTP represents the lambda router.
type HTTP struct {
	App *web.App
}

// APIGWHandler routes the lambda request (proxied from the API GW) to an internal endpoint.
func (l HTTP) APIGWHandler(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var ra lambdaproxy.RequestAccessorV2

	r, err := ra.EventToRequestWithContext(ctx, request)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{}, fmt.Errorf("event to request: %w", err)
	}

	w := lambdaproxy.NewProxyResponseWriterV2()
	l.App.ServeHTTP(w, r)

	return w.GetProxyResponse()
}

// ALBHandler routes the lambda request (proxied from the ALB) to an internal endpoint.
func (l HTTP) ALBHandler(ctx context.Context, request events.ALBTargetGroupRequest) (events.ALBTargetGroupResponse, error) {
	var ra lambdaproxy.RequestAccessorALB

	r, err := ra.EventToRequestWithContext(ctx, request)
	if err != nil {
		return events.ALBTargetGroupResponse{}, fmt.Errorf("ra.EventToRequestWithContext: %w", err)
	}

	w := lambdaproxy.NewProxyResponseWriterALB()
	l.App.ServeHTTP(w, r)

	evt, err := w.GetProxyResponse()
	if err != nil {
		return events.ALBTargetGroupResponse{}, fmt.Errorf("w.GetProxyResponse: %w", err)
	}

	return evt, nil
}
