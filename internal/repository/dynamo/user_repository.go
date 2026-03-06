// Copyright 2026 kpp.dev
//
// This file is part of kpfc.
//
// kpfc is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// kpfc is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with kpfc. If not, see <https://www.gnu.org/licenses/>.

package dynamo

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"kpp.dev/kpfc/internal/domain"
)

type UserRepository struct {
	client *dynamodb.Client
	table  string
}

func NewUserRepository(client *dynamodb.Client, tableName string) *UserRepository {
	return &UserRepository{
		client: client,
		table:  tableName,
	}
}

func (r *UserRepository) Create(user *domain.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	_, err = r.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName:           aws.String(r.table),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(id)"),
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(id string) (*domain.User, error) {
	result, err := r.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if result.Item == nil {
		return nil, domain.ErrNotFound
	}

	var user domain.User
	err = attributevalue.UnmarshalMap(result.Item, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*domain.User, error) {
	result, err := r.client.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:        aws.String(r.table),
		FilterExpression: aws.String("email = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: email},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan users by email: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, domain.ErrNotFound
	}

	var user domain.User
	err = attributevalue.UnmarshalMap(result.Items[0], &user)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Update(user *domain.User) error {
	item, err := attributevalue.MarshalMap(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	_, err = r.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

func (r *UserRepository) Delete(id string) error {
	_, err := r.client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
