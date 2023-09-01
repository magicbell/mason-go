// Package awslocal providers helpers for working with local aws services like dynamodb
package awslocal

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type localResolver struct {
	Host string
	Port string
}

func (r localResolver) ResolveEndpoint(service string, region string, options ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{URL: ("http://" + r.Host + r.Port)}, nil
}

func resolveWithLocalPort(host string, port string) aws.EndpointResolverWithOptions {
	return localResolver{
		Port: port,
		Host: host,
	}
}

func NewAWSLocalCfg(host string, port string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.Background(),
		config.WithRegion("eu-west-1"),
		config.WithEndpointResolverWithOptions(resolveWithLocalPort(host, port)),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SessionToken:    "test",
				Source:          "hardcoded_test_credentials",
			},
		}))
}
