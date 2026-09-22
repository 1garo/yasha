package account

import "github.com/google/uuid"

type AccountRequest struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Currency  string `json:"currency" validate:"required"`
}
type Account struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Currency  string    `json:"currency"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt,omitempty"`
	Balance   int64     `json:"balance"`
}

type BalanceResponse struct {
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency,omitempty"`
}
