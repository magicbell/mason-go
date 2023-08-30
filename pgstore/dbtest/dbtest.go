// Package dbtest contains supporting code for running tests that hit the DB.
package dbtest

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/magicbell-io/gofoundation/ddb"
	"github.com/ory/dockertest"
	redis "gopkg.in/redis.v5"
)

func StartPG() (*dockertest.Resource, error) {
	image := "postgres"
	tag := "13-alpine"
	port := "5432"
	dbName := "postgres"

	args := []string{"-e", "POSTGRES_PASSWORD=postgres", "-e", fmt.Sprintf("POSTGRES_DB=%s", dbName), "-e", fmt.Sprintf("POSTGRES_PORT=%s", port), "-p", "5432:5432"}

	d, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = d.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	pgResource, err := d.Run(image, tag, args)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	ping := func() error {
		conn, err := pgx.Connect(context.Background(), getDSN(dbName, pgResource.GetPort("5432/tcp")))
		if err != nil {
			return fmt.Errorf("pool.Acquire: %w", err)
		}
		defer conn.Close(context.Background())

		return nil
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := d.Retry(ping); err != nil {
		return pgResource, fmt.Errorf("pgstore in docker failed to ping: %w", err)
	}

	return pgResource, nil
}

func StartDDB() (*dockertest.Resource, *dynamodbstreams.Client, *dynamodb.Client, error) {
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

		cfg, err := NewAWSLocalCfg(port)
		if err != nil {
			return fmt.Errorf("unable to load SDK config, %v", err)
		}
		ddbClient = dynamodb.NewFromConfig(cfg)
		ddbStreamsClient = dynamodbstreams.NewFromConfig(cfg)

		err = ddb.CreateTable(ddbClient, tableName)
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

func StartRDB() (*dockertest.Resource, error) {
	image := "redis"
	tag := "6.2.6-alpine"
	args := []string{"-p", "6379:6379"}

	d, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = d.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	rdbResource, err := d.Run(image, tag, args)
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	ping := func() error {
		port := rdbResource.GetPort("6379/tcp")

		rdb := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("localhost:%s", port),
		})
		defer rdb.Close()

		if rdb.Ping().Err() != nil {
			return fmt.Errorf("unable to ping redis: %w", err)
		}

		return nil
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := d.Retry(ping); err != nil {
		return nil, fmt.Errorf("pgstore in docker failed to ping: %w", err)
	}

	return rdbResource, nil
}

func getDSN(dbName, dbPort string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:%s/%s?sslmode=disable", dbPort, dbName)
}

type localResolver struct {
	Port string
}

func (r localResolver) ResolveEndpoint(service string, region string, options ...interface{}) (aws.Endpoint, error) {
	return aws.Endpoint{URL: (`http://localhost:` + r.Port)}, nil
}

func resolveWithLocalPort(port string) aws.EndpointResolverWithOptions {
	return localResolver{
		Port: port,
	}
}

func NewAWSLocalCfg(port string) (aws.Config, error) {
	return config.LoadDefaultConfig(context.Background(),
		config.WithRegion("eu-west-1"),
		config.WithEndpointResolverWithOptions(resolveWithLocalPort(port)),
		config.WithCredentialsProvider(credentials.StaticCredentialsProvider{
			Value: aws.Credentials{
				AccessKeyID:     "test",
				SecretAccessKey: "test",
				SessionToken:    "test",
				Source:          "hardcoded_test_credentials",
			},
		}))
}
