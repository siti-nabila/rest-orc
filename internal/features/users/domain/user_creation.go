package domain

import "context"

const UserCreatePermission = "iam.user.create"

type UserCreationActor struct {
	UserID             uint64
	MustChangePassword bool
	Permissions        map[string]bool
}

// Pointer presence distinguishes omitted/null optional inputs from selected empty strings.
type CreateUserBody struct {
	Email       *string `json:"email"`
	Password    *string `json:"password"`
	Name        *string `json:"name"`
	Phone       *string `json:"phone"`
	CountryCode *string `json:"country_code"`
	Address     *string `json:"address"`
	Status      *string `json:"status"`
}
type CreateUserRequest struct {
	Body           CreateUserBody
	IdempotencyKey string
}
type CreateUserResponse struct {
	User *UserDetail `json:"user"`
}
type CreateUserGateway interface {
	CreateUser(context.Context, CreateUserRequest) (CreateUserResponse, error)
}
