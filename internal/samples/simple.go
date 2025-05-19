package samples

/*
type BadlyAlignedUser struct {
	IsActive  bool      // 1 byte
	Name      string    // 16 bytes (8 for pointer, 8 for length)
	Age       int8      // 1 byte
	Scores    []float64 // 24 bytes (8 for pointer, 8 for length, 8 for capacity)
	ID        uint16    // 2 bytes
	CreatedAt time.Time // 24 bytes
	Email     string    // 16 bytes
	IsAdmin   bool      // 1 byte
}
*/

/*
type ComplicatedUserProfile struct {
	UserID          uint32 // 4 bytes
	Activated       bool   // 1 byte
	Username        string // 16 bytes
	PersonalDetails struct {
		FirstName     string // 16 bytes
		MiddleInitial byte   // 1 byte
		LastName      string // 16 bytes
		Age           int8   // 1 byte
	}
	SessionToken       [16]byte          // 16 bytes fixed array
	LastLoginTime      time.Time         // 24 bytes
	PreferenceFlags    uint8             // 1 byte
	AccountBalance     float64           // 8 bytes
	Friends            map[string]uint64 // 8 bytes (pointer to map)
	RecentSearches     []string          // 24 bytes (slice)
	PremiumMember      bool              // 1 byte
	MemberSince        time.Time         // 24 bytes
	NotificationPrefs  byte              // 1 byte
	ProfilePictureData []byte            // 24 bytes (slice)
	DeviceID           uint64            // 8 bytes
	EmailVerified      bool              // 1 byte
	AddressInfo        struct {
		Street    string // 16 bytes
		City      string // 16 bytes
		ZipCode   uint16 // 2 bytes
		Country   string // 16 bytes
		IsPrimary bool   // 1 byte
	}
	LoginAttempts     uint16            // 2 bytes
	SecurityQuestions [3]string         // 48 bytes (array of 3 strings)
	AccountType       byte              // 1 byte
	Preferences       map[string]string // 8 bytes (pointer to map)
}
*/
