package samples

import (
	"time"
)

// Person represents a simple person with basic information
type Person struct {
	ID        int64
	Name      string
	Email     string
	BirthDate time.Time
	Active    bool
}

// Address represents a physical address
type Address struct {
	Street  string
	City    string
	State   string
	ZipCode string
	Country string
}

// Customer extends Person with additional information
type Customer struct {
	Person
	CustomerID    string
	Address       Address
	AccountNumber int64
	Balance       float64
	LastPurchase  time.Time
	Tags          []string
	Preferences   map[string]string
}

// BadlyOrdered is a struct with fields not ordered optimally for memory layout
type BadlyOrdered struct {
	IsApproved bool       // 1 byte
	Balance    int64      // 8 bytes
	SHA        byte       // 1 byte
	ID         int32      // 4 bytes
	Confirmed  bool       // 1 byte
	Complex    complex128 // 16 bytes
	Eight      byte       // 1 byte
}

// OptimallyOrdered is the same struct with fields ordered to minimize padding
type OptimallyOrdered struct {
	F complex128 // 16 bytes
	B int64      // 8 bytes
	D int32      // 4 bytes
	A bool       // 1 byte
	C byte       // 1 byte
	E bool       // 1 byte
	G byte       // 1 byte
}
