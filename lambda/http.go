// Package lambda provides support for running the app inside a Lambda function.
package lambda

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	lambdaproxy "github.com/awslabs/aws-lambda-go-api-proxy/core"
)

type HTTP struct {
	App http.Handler
}

// APIGWHandler routes the lambda request (proxied from the API GW) to an internal endpoint.
// Function URLs use the same format
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

	// Unescape RawQuery to undo the forced escaping by the `aws-lambda-go-api-proxy`. The forced escaping leads to
	// double encoding (once by the client, once by alb) issues. Which in practice means that a `something=key:value`
	// is encoded to `something=key%3Avalue` by the browser, and to `something=key%253Avalue` by ALB.
	// see: https://github.com/awslabs/aws-lambda-go-api-proxy/blob/55b777941b8d253f60a7aecf355cfd0a64f89dd7/core/requestALB.go#L135-L146
	if r.URL != nil && r.URL.RawQuery != "" && isLikelyDoubleEscaped(r.URL.RawQuery) {
		if unescaped, err := url.QueryUnescape(r.URL.RawQuery); err == nil {
			r.URL.RawQuery = unescaped
		}
	}

	w := lambdaproxy.NewProxyResponseWriterALB()
	l.App.ServeHTTP(w, r)

	evt, err := w.GetProxyResponse()
	if err != nil {
		return events.ALBTargetGroupResponse{}, fmt.Errorf("w.GetProxyResponse: %w", err)
	}

	return evt, nil
}

var doubleEncodedRE = regexp.MustCompile(`%25[0-9A-Fa-f]{2}`)

func isLikelyDoubleEscaped(raw string) bool {
	return doubleEncodedRE.MatchString(raw)
}
