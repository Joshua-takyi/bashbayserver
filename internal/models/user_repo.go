package models

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
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

	// define a small struct matching the DB row we selected
	type userRow struct {
		ID          string            `json:"id"`
		Email       string            `json:"email"`
		Username    string            `json:"username"`
		FullName    string            `json:"fullname"`
		Role        string            `json:"role"`
		Location    string            `json:"location"`
		Bio         string            `json:"bio"`
		Preferences map[string]string `json:"preferences"`
		PhoneNumber string            `json:"phone_number"`
		IsVerified  bool              `json:"is_verified"`
		AvatarURL   string            `json:"avatar_url"`
		CreatedAt   time.Time         `json:"created_at"`
		UpdatedAt   time.Time         `json:"updated_at"`
	}

	// Supabase returns an array even for single results
	var rows []userRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user rows: %v", err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	if len(rows) > 1 {
		return nil, fmt.Errorf("multiple users found for ID %s", stringedId)
	}

	row := rows[0]
	uid, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id from db: %v", err)
	}

	user := &User{
		ID:          uid,
		Username:    row.Username,
		FullName:    row.FullName,
		Email:       row.Email,
		IsVerified:  row.IsVerified,
		Bio:         row.Bio,
		Role:        row.Role,
		Location:    row.Location,
		Preferences: row.Preferences,
		PhoneNumber: row.PhoneNumber,
		AvatarURL:   row.AvatarURL,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}

	return user, nil
}

func (su *SupabaseRepo) UpdateUser(ctx context.Context, user *User) error {
	// TODO: Implement UpdateUser for Supabase
	return fmt.Errorf("UpdateUser not implemented yet")
}

func (su *SupabaseRepo) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// TODO: Implement DeleteUser for Supabase
	return fmt.Errorf("DeleteUser not implemented yet")
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
