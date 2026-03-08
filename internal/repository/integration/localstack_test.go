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

package integration

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const localstackEndpoint = "http://localhost:4566"

// checkLocalStackAvailable verifica se o LocalStack está rodando via conexão TCP
func checkLocalStackAvailable() bool {
	conn, err := net.DialTimeout("tcp", "localhost:4566", 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// getAWSConfig retorna configuração AWS para LocalStack
func getAWSConfig(t *testing.T) aws.Config {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	require.NoError(t, err)
	return cfg
}

func TestE2E_LocalStackHealthCheck(t *testing.T) {
	if !checkLocalStackAvailable() {
		t.Skip("LocalStack not available at localhost:4566")
	}

	cfg := getAWSConfig(t)
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(localstackEndpoint)
	})

	// Tentar listar tabelas como health check
	ctx := context.Background()
	_, err := client.ListTables(ctx, &dynamodb.ListTablesInput{})

	assert.NoError(t, err, "LocalStack DynamoDB should be accessible")
}

func TestE2E_DynamoDBUserCRUD(t *testing.T) {
	if !checkLocalStackAvailable() {
		t.Skip("LocalStack not available at localhost:4566")
	}

	cfg := getAWSConfig(t)
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(localstackEndpoint)
	})

	ctx := context.Background()

	// Criar usuário
	userID := "test-user-1"
	email := "test@example.com"

	putInput := &dynamodb.PutItemInput{
		TableName: aws.String("kpfc_users"),
		Item: map[string]types.AttributeValue{
			"id":       &types.AttributeValueMemberS{Value: userID},
			"email":    &types.AttributeValueMemberS{Value: email},
			"name":     &types.AttributeValueMemberS{Value: "Test User"},
			"provider": &types.AttributeValueMemberS{Value: "local"},
		},
	}

	_, err := client.PutItem(ctx, putInput)
	require.NoError(t, err, "should create user in DynamoDB")

	// Ler usuário
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String("kpfc_users"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := client.GetItem(ctx, getInput)
	require.NoError(t, err, "should read user from DynamoDB")
	assert.NotNil(t, result.Item)

	// Verificar dados
	emailAttr, ok := result.Item["email"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, email, emailAttr.Value)

	// Atualizar usuário
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String("kpfc_users"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: userID},
		},
		UpdateExpression: aws.String("SET #name = :name"),
		ExpressionAttributeNames: map[string]string{
			"#name": "name",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name": &types.AttributeValueMemberS{Value: "Updated Name"},
		},
	}

	_, err = client.UpdateItem(ctx, updateInput)
	assert.NoError(t, err, "should update user in DynamoDB")

	// Deletar usuário
	deleteInput := &dynamodb.DeleteItemInput{
		TableName: aws.String("kpfc_users"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: userID},
		},
	}

	_, err = client.DeleteItem(ctx, deleteInput)
	assert.NoError(t, err, "should delete user from DynamoDB")

	// Verificar que foi deletado
	getResult, err := client.GetItem(ctx, getInput)
	require.NoError(t, err)
	assert.Empty(t, getResult.Item, "user should be deleted")
}

func TestE2E_DynamoDBDeckCRUD(t *testing.T) {
	if !checkLocalStackAvailable() {
		t.Skip("LocalStack not available at localhost:4566")
	}

	cfg := getAWSConfig(t)
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(localstackEndpoint)
	})

	ctx := context.Background()

	// Criar deck
	deckID := "test-deck-1"
	userID := "test-user-1"

	putInput := &dynamodb.PutItemInput{
		TableName: aws.String("kpfc_decks"),
		Item: map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: deckID},
			"user_id":     &types.AttributeValueMemberS{Value: userID},
			"title":       &types.AttributeValueMemberS{Value: "Test Deck"},
			"description": &types.AttributeValueMemberS{Value: "A test deck"},
			"is_public":   &types.AttributeValueMemberBOOL{Value: true},
		},
	}

	_, err := client.PutItem(ctx, putInput)
	require.NoError(t, err, "should create deck in DynamoDB")

	// Ler deck
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String("kpfc_decks"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: deckID},
		},
	}

	result, err := client.GetItem(ctx, getInput)
	require.NoError(t, err, "should read deck from DynamoDB")
	assert.NotNil(t, result.Item)

	titleAttr, ok := result.Item["title"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "Test Deck", titleAttr.Value)

	// Deletar deck
	deleteInput := &dynamodb.DeleteItemInput{
		TableName: aws.String("kpfc_decks"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: deckID},
		},
	}

	_, err = client.DeleteItem(ctx, deleteInput)
	assert.NoError(t, err, "should delete deck from DynamoDB")
}

func TestE2E_DynamoDBCardCRUD(t *testing.T) {
	if !checkLocalStackAvailable() {
		t.Skip("LocalStack not available at localhost:4566")
	}

	cfg := getAWSConfig(t)
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(localstackEndpoint)
	})

	ctx := context.Background()

	// Criar card
	cardID := "test-card-1"
	deckID := "test-deck-1"

	putInput := &dynamodb.PutItemInput{
		TableName: aws.String("kpfc_cards"),
		Item: map[string]types.AttributeValue{
			"id":          &types.AttributeValueMemberS{Value: cardID},
			"deck_id":     &types.AttributeValueMemberS{Value: deckID},
			"front":       &types.AttributeValueMemberS{Value: "Front text"},
			"back":        &types.AttributeValueMemberS{Value: "Back text"},
			"interval":    &types.AttributeValueMemberN{Value: "1"},
			"ease_factor": &types.AttributeValueMemberN{Value: "2.5"},
		},
	}

	_, err := client.PutItem(ctx, putInput)
	require.NoError(t, err, "should create card in DynamoDB")

	// Ler card
	getInput := &dynamodb.GetItemInput{
		TableName: aws.String("kpfc_cards"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: cardID},
		},
	}

	result, err := client.GetItem(ctx, getInput)
	require.NoError(t, err, "should read card from DynamoDB")
	assert.NotNil(t, result.Item)

	frontAttr, ok := result.Item["front"].(*types.AttributeValueMemberS)
	require.True(t, ok)
	assert.Equal(t, "Front text", frontAttr.Value)

	// Atualizar card (simular review)
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String("kpfc_cards"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: cardID},
		},
		UpdateExpression: aws.String("SET #interval = :interval"),
		ExpressionAttributeNames: map[string]string{
			"#interval": "interval",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":interval": &types.AttributeValueMemberN{Value: "6"},
		},
	}

	_, err = client.UpdateItem(ctx, updateInput)
	assert.NoError(t, err, "should update card interval after review")

	// Deletar card
	deleteInput := &dynamodb.DeleteItemInput{
		TableName: aws.String("kpfc_cards"),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: cardID},
		},
	}

	_, err = client.DeleteItem(ctx, deleteInput)
	assert.NoError(t, err, "should delete card from DynamoDB")
}
