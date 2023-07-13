package models

type WithdrawDTO struct {
	Order        string  `json:"order"`
	Sum          float64 `json:"sum"`
	Processed_at string  `json:"processed_at"`
}
