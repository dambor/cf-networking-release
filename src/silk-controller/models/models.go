package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Route struct {
	Subnet string `json:"subnet,omitempty"`
	VtepIP string `json:"vtep_ip"`
}

type NewLeaseRequest struct {
	VtepIP string `json:"vtep_ip"`
}

type Lease struct {
	Route
	ID          string     `json:"id"`
	LastRenewed *time.Time `json:"last_renewed,omitempty"`
}

type AcquireLeaseResponse struct {
	Self   Lease   `json:"self"`
	Routes []Route `json:"routes"`
}

func NewID() (string, error) {
	const nBytes = 16
	bytes := make([]byte, nBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("reading random bytes: %s", err)
	}
	return hex.EncodeToString(bytes), nil
}
