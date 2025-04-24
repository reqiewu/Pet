package model

type User struct {
	ID         int64  `json:"id"`
	UserName   string `json:"username"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Phone      string `json:"phone"`
	UserStatus int32  `json:"userStatus"`
}

type LoginRequest struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}
type LoginResponse struct {
	Token string `json:"token"`
}
