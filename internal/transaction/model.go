package transaction

type TxType string

const (
	TxTypeDeposit  TxType = "deposit"
	TxTypeWithdraw TxType = "withdraw"
	TxTypeTransfer TxType = "transfer"
)

func (t TxType) IsValid() bool {
	return t == TxTypeDeposit || t == TxTypeWithdraw || t == TxTypeTransfer
}

type TransactionRequest struct {
	AccountID   string `json:"accountId" validate:"required"`
	Type        TxType `json:"type" validate:"required"`
	AmountMinor int64  `json:"amountMinor" validate:"required"`
	Currency    string `json:"currency" validate:"required"`
	To          string `json:"to,omitempty"`
}
