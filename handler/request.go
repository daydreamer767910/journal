package handler

type jsonHTTPLoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
	//RememberMe bool   `json:"rememberMe"`
}

type jsonHTTPVerify2FA struct {
	Code string `json:"code"`
}

type jsonHTTPRegisterForm struct {
	Username string `json:"username" validate:"required,alphanum,min=3"`
	Password string `json:"password" validate:"required,min=4"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone" validate:"numeric"`
	Adress   string `json:"address"`
}

type jsonHTTPDeleteFiles struct {
	Files []string `json:"files"`
}

// ChangePasswordRequest 结构定义了修改密码的请求参数
type jsonHTTPChangePassword struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword" validate:"required,min=4"`
}
