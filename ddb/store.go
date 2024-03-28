// Package ddbstore provides DynamoDB related functions for the store.
package ddb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodbstreams"
)

// DDBStore represents the DynamoDB store in the application.
type Store struct {
	client    *dynamodb.Client
	tableName *string
}

// New constructs a DynamoDB store.
func NewStore(client *dynamodb.Client, streamClient *dynamodbstreams.Client, tableName *string) *Store {
	return &Store{
		client:    client,
		tableName: tableName,
	}
}

type Item interface {
	GetType() string
}

func (s *Store) Save(ctx context.Context, item Item) error {
	ddbItem, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("av.MarshalMap: %w", err)
	}

	if _, ok := (ddbItem["CreatedAt"]).(*types.AttributeValueMemberS); !ok {
		ddbItem["CreatedAt"] = &types.AttributeValueMemberS{
			Value: time.Now().UTC().Format(time.RFC3339Nano),
		}
	}
	ddbItem["UpdatedAt"] = &types.AttributeValueMemberS{
		Value: time.Now().UTC().Format(time.RFC3339Nano),
	}
	ddbItem["Type"] = &types.AttributeValueMemberS{
		Value: item.GetType(),
	}

	input := dynamodb.PutItemInput{
		TableName: s.tableName,
		Item:      ddbItem,
	}

	_, err = s.client.PutItem(ctx, &input)
	if err != nil {
		return fmt.Errorf("ddb.PutItem: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, input *dynamodb.QueryInput) ([]map[string]types.AttributeValue, error) {
	input.TableName = s.tableName
	out, err := s.client.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("ddb.QueryPages: %w", err)
	}

	return out.Items, nil
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

	return out.Item, nil
}

func (s *Store) Discard(ctx context.Context, pk string, sk string) error {
	_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: s.tableName,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{
				Value: pk,
			},
			"SK": &types.AttributeValueMemberS{
				Value: sk,
			},
		},
		UpdateExpression:    aws.String("SET DiscardedAt = :discardedAt"),
		ConditionExpression: aws.String("attribute_exists(PK) AND attribute_exists(SK)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":discardedAt": &types.AttributeValueMemberS{
				Value: time.Now().UTC().Format(time.RFC3339Nano),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("ddb.DiscardItem: %w", err)
	}

	return nil
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
