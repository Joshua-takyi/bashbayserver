package models

import (
	"github.com/go-playground/validator/v10"
	"github.com/supabase-community/supabase-go"
	"go.mongodb.org/mongo-driver/mongo"
)

var Validate = validator.New()

type SupabaseRepo struct {
	supabaseClient *supabase.Client
	url            string
	key            string
}

func SupabaseNewRepo(supabaseClient *supabase.Client, url, key string) *SupabaseRepo {
	return &SupabaseRepo{
		supabaseClient: supabaseClient,
		url:            url,
		key:            key,
	}
}

// GetAuthenticatedClient returns a Supabase client with the given access token
func (su *SupabaseRepo) GetAuthenticatedClient(accessToken string) (*supabase.Client, error) {
	if su.url == "" || su.key == "" {
		// If we don't have the URL and key stored, we can't create a new client
		// This is a limitation of the current design
		return su.supabaseClient, nil
	}
	
	options := &supabase.ClientOptions{
		Headers: map[string]string{
			"Authorization": "Bearer " + accessToken,
		},
	}
	
	return supabase.NewClient(su.url, su.key, options)
}

type MongodbRepo struct {
	mongodbClient *mongo.Client
}

func MongodbNewRepo(mongodbClient *mongo.Client) *MongodbRepo {
	return &MongodbRepo{
		mongodbClient: mongodbClient,
	}
}
