package helpers

type EnhancedClaims struct {
	*CustomClaims
	Role        string `json:"role"`
	UserID      string `json:"id"`
	Email       string `json:"email,omitempty"`
	Username    string `json:"username,omitempty"`
	Fullname    string `json:"fullname,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
}

// Helper methods for role checking
func (ec *EnhancedClaims) IsAdmin() bool {
	return ec.Role == "admin"
}

func (ec *EnhancedClaims) IsHost() bool {
	return ec.Role == "host"
}

func (ec *EnhancedClaims) HasRole(role string) bool {
	return ec.Role == role
}

func (ec *EnhancedClaims) IsOwner(userID string) bool {
	return ec.UserID == userID
}

func (ec *EnhancedClaims) GetSafeRole() string {
	if ec.Role == "" {
		return "guest"
	}
	return ec.Role
}
