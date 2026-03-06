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
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"kpp.dev/kpfc/internal/domain"
)

type CardRepository struct {
	client *dynamodb.Client
	table  string
}

func NewCardRepository(client *dynamodb.Client, tableName string) *CardRepository {
	return &CardRepository{
		client: client,
		table:  tableName,
	}
}

func (r *CardRepository) Create(card *domain.Card) error {
	item, err := attributevalue.MarshalMap(card)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	_, err = r.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create card: %w", err)
	}

	return nil
}

func (r *CardRepository) GetByID(id string) (*domain.Card, error) {
	result, err := r.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	if result.Item == nil {
		return nil, domain.ErrNotFound
	}

	var card domain.Card
	err = attributevalue.UnmarshalMap(result.Item, &card)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal card: %w", err)
	}

	return &card, nil
}

func (r *CardRepository) GetByDeckID(deckID string) ([]*domain.Card, error) {
	result, err := r.client.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:        aws.String(r.table),
		FilterExpression: aws.String("deck_id = :deck_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":deck_id": &types.AttributeValueMemberS{Value: deckID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan cards by deck: %w", err)
	}

	cards := make([]*domain.Card, 0, len(result.Items))
	for _, item := range result.Items {
		var card domain.Card
		err = attributevalue.UnmarshalMap(item, &card)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal card: %w", err)
		}
		cards = append(cards, &card)
	}

	return cards, nil
}

func (r *CardRepository) GetDueCards(deckID string) ([]*domain.Card, error) {
	allCards, err := r.GetByDeckID(deckID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	dueCards := make([]*domain.Card, 0)
	for _, card := range allCards {
		if card.DueDate.Before(now) || card.DueDate.Equal(now) {
			dueCards = append(dueCards, card)
		}
	}

	return dueCards, nil
}

func (r *CardRepository) Update(card *domain.Card) error {
	item, err := attributevalue.MarshalMap(card)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %w", err)
	}

	_, err = r.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}

	return nil
}

func (r *CardRepository) Delete(id string) error {
	_, err := r.client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	return nil
}
