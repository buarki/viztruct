package samples

import (
	"sync"
	"time"
)

// BadLayoutStruct is intentionally poorly laid out to demonstrate memory inefficiency
type BadLayoutStruct struct {
	ID           uint64
	IsActive     bool
	Name         string
	Age          uint8
	Score        float64
	LastLogin    time.Time
	IsVerified   bool
	Balance      float32
	Email        string
	PhoneNumber  string
	IsPremium    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Status       uint16
	IsBlocked    bool
	LoginCount   uint32
	LastIP       string
	IsSuspended  bool
	SessionToken [16]byte
	IsDeleted    bool
	ProfilePic   []byte
	IsLocked     bool
	FailedLogins uint8
	IsArchived   bool
	LastSync     time.Time
	IsHidden     bool
	Version      uint64
	IsPinned     bool
	LastBackup   time.Time
	IsStarred    bool
	CustomFlags  uint32
	IsFavorited  bool
	LastModified time.Time
	IsProtected  bool
	AccessLevel  uint8
	IsEncrypted  bool
	LastAccessed time.Time
	IsCompressed bool
	FileSize     int64
	wg           sync.WaitGroup
	IsIndexed    bool
}

// OptimizedLayoutStruct is optimized for memory efficiency with fields ordered by size and alignment
type OptimizedLayoutStruct struct {
	// 24-byte aligned fields first
	ProfilePic   []byte
	LastSync     time.Time
	LastBackup   time.Time
	UpdatedAt    time.Time
	CreatedAt    time.Time
	LastLogin    time.Time
	LastAccessed time.Time
	LastModified time.Time

	// 16-byte aligned fields
	Email       string
	PhoneNumber string
	Name        string
	LastIP      string

	// 12-byte field with 4-byte padding
	wg sync.WaitGroup

	// 8-byte aligned fields
	ID       uint64
	Score    float64
	FileSize int64
	Version  uint64

	// 4-byte aligned fields
	CustomFlags uint32
	Balance     float32
	LoginCount  uint32

	// 2-byte aligned field
	Status uint16

	// 1-byte aligned fields
	SessionToken [16]byte
	IsArchived   bool
	IsFavorited  bool
	IsLocked     bool
	IsHidden     bool
	IsBlocked    bool
	IsPinned     bool
	IsSuspended  bool
	IsStarred    bool
	IsPremium    bool
	FailedLogins uint8
	IsDeleted    bool
	IsProtected  bool
	AccessLevel  uint8
	IsEncrypted  bool
	IsVerified   bool
	IsCompressed bool
	Age          uint8
	IsActive     bool
	IsIndexed    bool
}
