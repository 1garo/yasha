package server

import (
	"github.com/1garo/yasha/internal/account"
	"github.com/1garo/yasha/internal/transaction"
)

func (s *server) RegisterRoutes() {
	accHandler := account.NewHandler(s.db)
	txHandler := transaction.NewHandler(s.db)

	accGroup := s.e.Group("/account")
	txGroup := s.e.Group("/transaction")

	txGroup.POST("", txHandler.TransactionHandler)

	accGroup.GET("/:id/balance", accHandler.BalanceHandler)
	accGroup.GET("/:id", accHandler.GetAccountHandler)
	accGroup.POST("", accHandler.CreateAccountHandler)
}
