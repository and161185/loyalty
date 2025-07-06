package model

import "time"

type OrderStatus string

const (
	New        OrderStatus = "NEW"
	Registered OrderStatus = "REGISTERED"
	Invalid    OrderStatus = "INVALID"
	Processing OrderStatus = "PROCESSING"
	Processed  OrderStatus = "PROCESSED"
)

type Balance struct {
	Current   float64
	Withdrawn float64
}

type Order struct {
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    *float64    `json:"accrual,omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
}

type Withdrawal struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

type User struct {
	ID    int
	Login string
}
