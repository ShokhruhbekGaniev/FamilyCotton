package model

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if r.Login == "" {
		return NewAppError(ErrValidation, "login is required")
	}
	if r.Password == "" {
		return NewAppError(ErrValidation, "password is required")
	}
	return nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return NewAppError(ErrValidation, "refresh_token is required")
	}
	return nil
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
