package models

type AccrualDTO struct {
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}
