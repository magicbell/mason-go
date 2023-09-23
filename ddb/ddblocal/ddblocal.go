// Package ddblocal provides support for running a DynamoDB instance in a Docker container.
package ddblocal

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/magicbell-io/mason-go/awslocal"
	"github.com/magicbell-io/mason-go/ddb"
	"github.com/ory/dockertest"
)

type Config struct {
	port  string
	tag   string
	image string
}

func NewConfig(image string, tag string, port string) *Config {
	return &Config{
		port:  port,
		tag:   tag,
		image: image,
	}
}

func (c *Config) Start() (*dockertest.Resource, *dynamodbstreams.Client, *dynamodb.Client, error) {
	image := "amazon/dynamodb-local"
	tag := "latest"
	args := []string{"-p", "8000:8000"}

	d, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = d.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	ddbResource, err := d.Run(image, tag, args)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	var ddbStreamsClient *dynamodbstreams.Client
	var ddbClient *dynamodb.Client
	ping := func() error {
		tableName := "ping-test"

		port := ddbResource.GetPort("8000/tcp")

		cfg, err := awslocal.NewConfig("localhost", port)
		if err != nil {
			return fmt.Errorf("awslocal.NewConfig: %v", err)
		}
		ddbClient = dynamodb.NewFromConfig(cfg)
		ddbStreamsClient = dynamodbstreams.NewFromConfig(cfg)

		admin := ddb.NewAdmin(ddbClient)
		err = admin.CreateTable(tableName)
		if err != nil {
			return fmt.Errorf("CreateTable: %w", err)
		}

		defer func() {
			ddbClient.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
				TableName: &tableName,
			})
		}()

		return nil
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := d.Retry(ping); err != nil {
		return ddbResource, ddbStreamsClient, ddbClient, fmt.Errorf("pgstore in docker failed to ping: %w", err)
	}

	return ddbResource, ddbStreamsClient, ddbClient, nil
}
