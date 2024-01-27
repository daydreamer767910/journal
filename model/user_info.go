package model

type User struct {
	Userid   string `json:"userid"`
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"emal"`
	Phone    string `json:"phone"`
	Adress   string `json:"address"`
	// PasswordHash takes precedence over Password.
	PasswordHash string `json:"password_hash"`
	Admin        bool   `json:"admin"`
	// Token for JWT verification
	Token string `json:"token"`
	// 2FA
	Secret2FA string `json:"secret2fa"`
	Enable2FA bool   `json:"enable2fa"`
}
