package dto

type RegisterRequest struct {
	FullName        string `json:"full_name"        binding:"required,min=2,max=100"`
	Email           string `json:"email"            binding:"required,email"`
	Password        string `json:"password"         binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

type LoginRequest struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type GoogleLoginRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Email              string `json:"email" binding:"required,email"`
	OTP                string `json:"otp" binding:"required,len=6,numeric"`
	NewPassword        string `json:"new_password" binding:"required,min=8"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required,eqfield=NewPassword"`
}



type MessageResponse struct {
	Message string `json:"message"`
}
