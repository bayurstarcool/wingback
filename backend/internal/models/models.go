// Package models holds the canonical domain types shared between layers.
// Keep this package free of transport-specific tags (json/validate) —
// those belong in handler/DTO layers.
package models

import "time"

type User struct {
	ID                    string
	Email                 string
	Username              string
	PasswordHash          string
	DisplayName           string
	AvatarURL             string
	Currency              int
	LastLat               *float64
	LastLng               *float64
	LastLocationAt        *time.Time
	LastLocationAccuracyM *float64
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type Carrier struct {
	ID        string
	Slug      string
	Name      string
	SpeedKMH  float64
	IsDefault bool
	Price     int
	Rarity    string
	AssetURL  string
}

type MessageStatus string

const (
	StatusInTransit MessageStatus = "in_transit"
	StatusDelivered MessageStatus = "delivered"
	StatusLost      MessageStatus = "lost"
)

type Message struct {
	ID               string
	SenderID         string
	RecipientID      string
	CarrierID        string
	Body             string
	SenderLat        float64
	SenderLng        float64
	RecLat           float64
	RecLng           float64
	DistanceKM       float64
	SpeedKMH         float64
	Status           MessageStatus
	DepartsAt        time.Time
	ArrivesAt        time.Time
	DeliveredAt      *time.Time
	SpeedupsUsed     int
	LocationPrivacy  string
	SenderCity       string
	RecipientCity    string
	SenderCityLat    *float64
	SenderCityLng    *float64
	RecipientCityLat *float64
	RecipientCityLng *float64
	CreatedAt        time.Time
}

const (
	LocationPrivacyAccurate = "accurate"
	LocationPrivacyHidden   = "hidden"
)
