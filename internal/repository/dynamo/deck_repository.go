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
	"github.com/google/uuid"

	"kpp.dev/kpfc/internal/domain"
)

type DeckRepository struct {
	client         *dynamodb.Client
	table          string
	cardRepository *CardRepository
}

func NewDeckRepository(client *dynamodb.Client, tableName string, cardRepo *CardRepository) *DeckRepository {
	return &DeckRepository{
		client:         client,
		table:          tableName,
		cardRepository: cardRepo,
	}
}

func (r *DeckRepository) Create(deck *domain.Deck) error {
	item, err := attributevalue.MarshalMap(deck)
	if err != nil {
		return fmt.Errorf("failed to marshal deck: %w", err)
	}

	_, err = r.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to create deck: %w", err)
	}

	return nil
}

func (r *DeckRepository) GetByID(id string) (*domain.Deck, error) {
	result, err := r.client.GetItem(context.Background(), &dynamodb.GetItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get deck: %w", err)
	}

	if result.Item == nil {
		return nil, domain.ErrNotFound
	}

	var deck domain.Deck
	err = attributevalue.UnmarshalMap(result.Item, &deck)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deck: %w", err)
	}

	return &deck, nil
}

func (r *DeckRepository) GetByUserID(userID string) ([]*domain.Deck, error) {
	result, err := r.client.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:        aws.String(r.table),
		FilterExpression: aws.String("user_id = :user_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":user_id": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan decks by user: %w", err)
	}

	decks := make([]*domain.Deck, 0, len(result.Items))
	for _, item := range result.Items {
		var deck domain.Deck
		err = attributevalue.UnmarshalMap(item, &deck)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal deck: %w", err)
		}
		decks = append(decks, &deck)
	}

	return decks, nil
}

func (r *DeckRepository) GetPublic() ([]*domain.Deck, error) {
	result, err := r.client.Scan(context.Background(), &dynamodb.ScanInput{
		TableName:        aws.String(r.table),
		FilterExpression: aws.String("is_public = :is_public"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":is_public": &types.AttributeValueMemberBOOL{Value: true},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to scan public decks: %w", err)
	}

	decks := make([]*domain.Deck, 0, len(result.Items))
	for _, item := range result.Items {
		var deck domain.Deck
		err = attributevalue.UnmarshalMap(item, &deck)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal deck: %w", err)
		}
		decks = append(decks, &deck)
	}

	return decks, nil
}

func (r *DeckRepository) Update(deck *domain.Deck) error {
	item, err := attributevalue.MarshalMap(deck)
	if err != nil {
		return fmt.Errorf("failed to marshal deck: %w", err)
	}

	_, err = r.client.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(r.table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to update deck: %w", err)
	}

	return nil
}

func (r *DeckRepository) Delete(id string) error {
	_, err := r.client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(r.table),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete deck: %w", err)
	}

	return nil
}

func (r *DeckRepository) Clone(deckID, newUserID string) (*domain.Deck, error) {
	originalDeck, err := r.GetByID(deckID)
	if err != nil {
		return nil, err
	}

	if !originalDeck.IsPublic {
		return nil, domain.ErrDeckNotPublic
	}

	newDeck := &domain.Deck{
		ID:          uuid.New().String(),
		UserID:      newUserID,
		Title:       originalDeck.Title,
		Description: originalDeck.Description,
		IsPublic:    false,
		CreatedAt:   originalDeck.CreatedAt,
		UpdatedAt:   originalDeck.UpdatedAt,
	}

	if err := r.Create(newDeck); err != nil {
		return nil, err
	}

	originalCards, err := r.cardRepository.GetByDeckID(deckID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cards from original deck: %w", err)
	}

	for _, card := range originalCards {
		newCard := &domain.Card{
			ID:         uuid.New().String(),
			DeckID:     newDeck.ID,
			Front:      card.Front,
			Back:       card.Back,
			Interval:   1,
			EaseFactor: 2.5,
			DueDate:    card.DueDate,
			CreatedAt:  card.CreatedAt,
			UpdatedAt:  card.UpdatedAt,
		}
		if err := r.cardRepository.Create(newCard); err != nil {
			return nil, fmt.Errorf("failed to clone card: %w", err)
		}
	}

	return newDeck, nil
}
