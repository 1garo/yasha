package account

import "github.com/google/uuid"

type AccountRequest struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}
type Account struct {
	ID        uuid.UUID         `json:"id"`
	FirstName string            `json:"firstName"`
	LastName  string            `json:"lastName"`
	Email     string            `json:"email"`
	CreatedAt string            `json:"createdAt"`
	UpdatedAt string            `json:"updatedAt,omitempty"`
	Balances  []BalanceResponse `json:"balances"`
}

type BalanceResponse struct {
	AmountMinor int64  `json:"amountMinor"`
	Currency    string `json:"currency,omitempty"`
}
