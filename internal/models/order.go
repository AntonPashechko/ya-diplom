package models

type OrderDTO struct {
	Number     string `json:"number"`
	Status     string `json:"status"`
	Accrual    int64  `json:"accrual,omitempty"`
	UploadedAt string `json:"uploaded_at"`
}
