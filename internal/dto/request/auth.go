package request

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Refresh struct {
	RefreshToken string `json:"refresh_token"`
}
