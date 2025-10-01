package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/supabase-community/gotrue-go/types"
)

const (
	ProfileTable = "profiles"
	EventsTable  = "events"
	VenuesTable  = "venues"
	DBName       = "rendez"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *User) (interface{}, error)
	AuthenticateUser(ctx context.Context, email, password string) (interface{}, error)
	RefreshToken(ctx context.Context, refreshToken string) (interface{}, error)
	GetUser(ctx context.Context, id uuid.UUID, accessToken string) (*User, error)
	UpdateUser(ctx context.Context, user map[string]interface{}, userid uuid.UUID, accessToken string) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID, accessToken string) error
	UploadAvatar(ctx context.Context, userId uuid.UUID, imageData string, accessToken string) (string, error)
}

func ConvertToUser(raw map[string]interface{}) (*User, error) {
	userBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw user: %v", err)
	}

	user := &User{}
	if err := json.Unmarshal(userBytes, user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to user struct: %v", err)
	}

	return user, nil
}

func (su *SupabaseRepo) CreateUser(ctx context.Context, user *User) (interface{}, error) {
	signed := types.SignupRequest{
		Email:    user.Email,
		Password: user.Password,
	}

	res, err := su.supabaseClient.Auth.Signup(signed)
	if err != nil {
		if strings.Contains(err.Error(), "User already Registered") {
			return nil, fmt.Errorf("email already in use")
		}

		// Parse and clean up database constraint errors
		errMsg := err.Error()
		if strings.Contains(errMsg, "null value in column") {
			if strings.Contains(errMsg, "username") {
				return nil, fmt.Errorf("username is required")
			}
			// Add other null constraint checks as needed
			return nil, fmt.Errorf("required field is missing")
		}

		// Parse other common database errors
		if strings.Contains(errMsg, "unique constraint") {
			return nil, fmt.Errorf("user already exists")
		}

		if strings.Contains(errMsg, "invalid input syntax") {
			return nil, fmt.Errorf("invalid input format")
		}

		// For any other errors, return a clean generic message
		return nil, fmt.Errorf("failed to create user")
	}
	return res, nil
}

func (su *SupabaseRepo) GetUser(ctx context.Context, id uuid.UUID, accessToken string) (*User, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("invalid UUID")
	}

	stringedId := id.String()

	// Use authenticated client if token is provided
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated client: %v", err)
		}
		client = authClient
	}

	raw, status, err := client.From(ProfileTable).
		Select("id,email,username,fullname,role,location,bio,preferences,phone_number,is_verified,avatar_url,created_at,updated_at", "", false).
		Eq("id", stringedId).
		// Single().
		Execute()
	if err != nil {
		// include response status and body when available so caller can distinguish
		if status != 0 {
			return nil, fmt.Errorf("postgrest error: status=%d body=%s err=%v", status, string(raw), err)
		}
		return nil, fmt.Errorf("failed to get user by ID: %v", err)
	}

	// Supabase returns an array even for single results
	var users []User
	if err := json.Unmarshal(raw, &users); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user rows: %v", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	if len(users) > 1 {
		return nil, fmt.Errorf("multiple users found for ID %s", stringedId)
	}

	return &users[0], nil
}

func (su *SupabaseRepo) UpdateUser(ctx context.Context, user map[string]interface{}, userid uuid.UUID, accessToken string) (*User, error) {

	if userid == uuid.Nil {
		return nil, fmt.Errorf("invalid UUID")
	}

	if len(user) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	client := su.supabaseClient

	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to create authenticated client: %v", err)
		}
		client = authClient
	}

	raw, count, err := client.From(ProfileTable).
		Update(user, "", "exact").
		Eq("id", userid.String()).
		Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	if count == 0 {
		return nil, fmt.Errorf("no user found to update %v", err)
	}

	var rawUsers []map[string]interface{}
	if err := json.Unmarshal(raw, &rawUsers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal updated user: %v", err)
	}

	if len(rawUsers) == 0 {
		return nil, fmt.Errorf("no user data returned after update")
	}

	updatedUser, err := ConvertToUser(rawUsers[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert updated user data: %v", err)
	}

	return updatedUser, nil
}

func (su *SupabaseRepo) DeleteUser(ctx context.Context, id uuid.UUID, accessToken string) error {
	if id == uuid.Nil {
		return fmt.Errorf("no valid UUID provided")
	}
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err != nil {
			return fmt.Errorf("failed to create authenticated client: %v", err)
		}
		client = authClient
	}

	raw, count, err := client.From(ProfileTable).Delete("", "exact").Eq("id", id.String()).Execute()

	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}

	if count == 0 {
		return fmt.Errorf("no user found to delete")
	}

	var rawUsers []map[string]interface{}
	if err := json.Unmarshal(raw, &rawUsers); err != nil {
		return fmt.Errorf("failed to unmarshal deleted user data: %v", err)
	}

	if len(rawUsers) == 0 {
		return fmt.Errorf("no user data returned after deletion")
	}

	// Optionally, convert and return the deleted user data if needed
	return nil
}

func (su *SupabaseRepo) AuthenticateUser(ctx context.Context, email, password string) (interface{}, error) {
	resp, err := su.supabaseClient.Auth.SignInWithEmailPassword(email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate user: %v", err)
	}
	return resp, nil
}

func (su *SupabaseRepo) RefreshToken(ctx context.Context, refreshToken string) (interface{}, error) {
	resp, err := su.supabaseClient.Auth.RefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %v", err)
	}
	return resp, nil
}

func (su *SupabaseRepo) UploadAvatar(ctx context.Context, userId uuid.UUID, imageData string, accessToken string) (string, error) {
	client := su.supabaseClient
	if accessToken != "" {
		authClient, err := su.GetAuthenticatedClient(accessToken)
		if err != nil {
			return "", fmt.Errorf("failed to create authenticated client: %v", err)
		}
		client = authClient
	}

	raw, count, err := client.From(ProfileTable).Update(map[string]interface{}{
		"avatar_url": imageData,
	}, "", "exact").Eq("id", userId.String()).Execute()
	if err != nil {
		return "", fmt.Errorf("failed to upload avatar: %v", err)
	}

	if count == 0 {
		return "", fmt.Errorf("no user found to update avatar")
	}

	var rawUsers []map[string]interface{}
	if err := json.Unmarshal(raw, &rawUsers); err != nil {
		return "", fmt.Errorf("failed to unmarshal updated user data: %v", err)
	}

	if len(rawUsers) == 0 {
		return "", fmt.Errorf("no user data returned after avatar upload")
	}

	updatedUser, err := ConvertToUser(rawUsers[0])
	if err != nil {
		return "", fmt.Errorf("failed to convert updated user data: %v", err)
	}

	return updatedUser.AvatarURL, nil
}
