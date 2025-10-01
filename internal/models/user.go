package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID         `db:"id" json:"id"`
	Username    string            `db:"username" json:"username"`
	FullName    string            `db:"fullname" json:"fullname"`
	Email       string            `db:"email" json:"email" `
	Password    string            `db:"password" json:"password" `
	IsVerified  bool              `db:"is_verified" json:"is_verified"`
	Bio         string            `db:"bio" json:"bio"`
	Role        string            `db:"role" json:"role"`
	Location    string            `db:"location" json:"location"`
	Preferences map[string]string `db:"preferences" json:"preferences"`
	PhoneNumber string            `db:"phone_number" json:"phone_number"`
	AvatarURL   string            `db:"avatar_url" json:"avatar_url"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`
}
