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

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server   ServerConfig
	JWT      JWTConfig
	AWS      AWSConfig
	DynamoDB DynamoDBConfig
	S3       S3Config
	OAuth    OAuthConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type JWTConfig struct {
	Secret     string
	Expiration string
}

type AWSConfig struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

type DynamoDBConfig struct {
	TableUsers string
	TableDecks string
	TableCards string
}

type S3Config struct {
	BucketAvatars string
}

type OAuthConfig struct {
	Google GoogleOAuthConfig
	GitHub GitHubOAuthConfig
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GitHubOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")

	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: port,
			Env:  env,
		},
		JWT: JWTConfig{
			Secret:     jwtSecret,
			Expiration: getEnv("JWT_EXPIRATION", "24h"),
		},
		AWS: AWSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			Endpoint:        getEnv("AWS_ENDPOINT", ""),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		DynamoDB: DynamoDBConfig{
			TableUsers: getEnv("DYNAMO_TABLE_USERS", "kpfc_users"),
			TableDecks: getEnv("DYNAMO_TABLE_DECKS", "kpfc_decks"),
			TableCards: getEnv("DYNAMO_TABLE_CARDS", "kpfc_cards"),
		},
		S3: S3Config{
			BucketAvatars: getEnv("S3_BUCKET_AVATARS", "kpfc-avatars"),
		},
		OAuth: OAuthConfig{
			Google: GoogleOAuthConfig{
				ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
				ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
			},
			GitHub: GitHubOAuthConfig{
				ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
				ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
				RedirectURL:  getEnv("GITHUB_REDIRECT_URL", ""),
			},
		},
		CORS: CORSConfig{
			AllowedOrigins: getSliceEnv("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.Split(value, ",")
}

func getIntEnv(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
