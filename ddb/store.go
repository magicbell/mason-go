// Package ddbstore provides DynamoDB related functions for the store.
package ddb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
	"go.uber.org/zap"
)

// DDBStore represents the DynamoDB store in the application.
type Store struct {
	log       *zap.SugaredLogger
	client    *dynamodb.Client
	tableName *string
}

// New constructs a DynamoDB store.
func NewStore(log *zap.SugaredLogger, client *dynamodb.Client, streamClient *dynamodbstreams.Client, tableName *string) *Store {
	return &Store{
		client:    client,
		log:       log,
		tableName: tableName,
	}
}

type coreItem struct {
	PK     string `dynamodbav:"PK"`
	SK     string `dynamodbav:"SK"`
	record interface{}
}

func (s *Store) Create(ctx context.Context, item map[string]types.AttributeValue) error {
	input := dynamodb.PutItemInput{
		TableName: s.tableName,
		Item:      item,
	}

	_, err := s.client.PutItem(ctx, &input)
	if err != nil {
		return fmt.Errorf("ddb.PutItem: %w", err)
	}

	return nil
}

func (s *Store) Fetch(ctx context.Context, pk string, sk string) (map[string]types.AttributeValue, error) {
	out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: s.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{
				Value: pk,
			},
			"SK": &types.AttributeValueMemberS{
				Value: sk,
			},
		},
	})
	if err != nil {
		return out.Item, err
	}

	s.log.Infow("ddb.GetItem", "out", out)

	return out.Item, nil
}

func (s *Store) Count(ctx context.Context) (int32, error) {
	input := &dynamodb.ScanInput{
		TableName: s.tableName,
		Select:    types.SelectCount,
	}
	resp, err := s.client.Scan(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to scan the table, %w", err)
	}
	return resp.Count, nil
}
