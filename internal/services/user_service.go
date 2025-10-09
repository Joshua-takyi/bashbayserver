package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/joshua-takyi/ww/internal/helpers"
	"github.com/joshua-takyi/ww/internal/models"
)

type UserService struct {
	userRepo models.UserRepo
}

func NewUserService(userRepo models.UserRepo) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (us *UserService) CreateUser(user *models.User) (interface{}, error) {
	if err := models.Validate.Struct(user); err != nil {
		return nil, err
	}

	ok := helpers.IsPasswordStrong(user.Password)
	if !ok {
		return nil, fmt.Errorf("password is not strong enough")
	}

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	return us.userRepo.CreateUser(context.Background(), user)
}

func (us *UserService) AuthenticateUser(email, password string) (interface{}, error) {
	if err := models.Validate.Var(email, "required,email"); err != nil {
		return nil, fmt.Errorf("invalid email format: %v", err)
	}
	if err := models.Validate.Var(password, "required,min=8"); err != nil {
		return nil, fmt.Errorf("invalid password format: %v", err)
	}
	response, err := us.userRepo.AuthenticateUser(context.Background(), email, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}

	return response, nil
}

func (us *UserService) RefreshToken(refreshToken string) (interface{}, error) {
	if refreshToken == "" {
		return nil, fmt.Errorf("refresh token is required")
	}
	response, err := us.userRepo.RefreshToken(context.Background(), refreshToken)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %v", err)
	}
	return response, nil
}

func (us *UserService) GetUser(id uuid.UUID, accessToken string) (*models.User, error) {
	res, err := us.userRepo.GetUser(context.Background(), id, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}
	return res, nil
}

func (us *UserService) UpdateUser(ctx context.Context, user map[string]interface{}, userid uuid.UUID, accessToken string) (*models.User, error) {
	// if err := models.Validate.Struct(user); err != nil {
	// 	return nil, err
	// }

	now := time.Now()
	user["updated_at"] = now

	updatedUser, err := us.userRepo.UpdateUser(ctx, user, userid, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %v", err)
	}

	return updatedUser, nil
}

func (us *UserService) DeleteUser(ctx context.Context, id uuid.UUID, accessToken string) error {
	err := us.userRepo.DeleteUser(ctx, id, accessToken)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", err)
	}
	return nil
}

func (su *UserService) UploadAvatar(ctx context.Context, userId uuid.UUID, imageData string, accessToken string) (string, error) {
	if userId == uuid.Nil {
		return "", fmt.Errorf("no valid UUID provided")
	}

	avatarURL, err := su.userRepo.UploadAvatar(ctx, userId, imageData, accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to upload avatar: %v", err)
	}

	return avatarURL, nil
}
