package structi

import (
	"runtime"
	"testing"
	"time"
	"unsafe"

	"github.com/buarki/viztruct/internal/samples"
)

func BenchmarkAdvancedMemoryUsage(b *testing.B) {
	const iterations = 100_000

	b.Run("BadLayoutStructs", func(b *testing.B) {
		b.ReportAllocs()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		beforeAlloc := m.TotalAlloc

		var items = make([]samples.BadLayoutStruct, iterations)
		for i := 0; i < iterations; i++ {
			items[i] = samples.BadLayoutStruct{
				ID:           uint64(i),
				IsActive:     true,
				Name:         "johndoe",
				Age:          uint8(i % 100),
				Score:        100.50,
				LastLogin:    time.Now(),
				IsVerified:   true,
				Balance:      1000.50,
				Email:        "john@example.com",
				PhoneNumber:  "1234567890",
				IsPremium:    true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Status:       uint16(i % 1000),
				IsBlocked:    false,
				LoginCount:   uint32(i),
				LastIP:       "192.168.1.1",
				IsSuspended:  false,
				SessionToken: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				IsDeleted:    false,
				ProfilePic:   []byte("profile picture data"),
				IsLocked:     false,
				FailedLogins: uint8(i % 5),
				IsArchived:   false,
				LastSync:     time.Now(),
				IsHidden:     false,
				Version:      uint64(i),
				IsPinned:     false,
				LastBackup:   time.Now(),
				IsStarred:    false,
				CustomFlags:  uint32(i),
				IsFavorited:  false,
				LastModified: time.Now(),
				IsProtected:  false,
				AccessLevel:  uint8(i % 10),
				IsEncrypted:  false,
				LastAccessed: time.Now(),
				IsCompressed: false,
				FileSize:     int64(i * 1000),
				IsIndexed:    false,
			}
		}

		runtime.ReadMemStats(&m)
		b.ReportMetric(float64(m.TotalAlloc-beforeAlloc)/float64(iterations), "B/op")

		structSize := unsafe.Sizeof(samples.BadLayoutStruct{})
		b.Logf("bad layout struct size: %d bytes", structSize)
		b.Logf("memory impact: bad layout uses %.2f MB per 1 million instances", float64(structSize)*1_000_000/1024/1024)
	})

	b.Run("OptimizedLayoutStructs", func(b *testing.B) {
		b.ReportAllocs()
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		beforeAlloc := m.TotalAlloc

		var items = make([]samples.OptimizedLayoutStruct, iterations)
		for i := 0; i < iterations; i++ {
			items[i] = samples.OptimizedLayoutStruct{
				ID:           uint64(i),
				IsActive:     true,
				Name:         "johndoe",
				Age:          uint8(i % 100),
				Score:        100.50,
				LastLogin:    time.Now(),
				IsVerified:   true,
				Balance:      1000.50,
				Email:        "john@example.com",
				PhoneNumber:  "1234567890",
				IsPremium:    true,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
				Status:       uint16(i % 1000),
				IsBlocked:    false,
				LoginCount:   uint32(i),
				LastIP:       "192.168.1.1",
				IsSuspended:  false,
				SessionToken: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				IsDeleted:    false,
				ProfilePic:   []byte("profile picture data"),
				IsLocked:     false,
				FailedLogins: uint8(i % 5),
				IsArchived:   false,
				LastSync:     time.Now(),
				IsHidden:     false,
				Version:      uint64(i),
				IsPinned:     false,
				LastBackup:   time.Now(),
				IsStarred:    false,
				CustomFlags:  uint32(i),
				IsFavorited:  false,
				LastModified: time.Now(),
				IsProtected:  false,
				AccessLevel:  uint8(i % 10),
				IsEncrypted:  false,
				LastAccessed: time.Now(),
				IsCompressed: false,
				FileSize:     int64(i * 1000),
				IsIndexed:    false,
			}
		}

		runtime.ReadMemStats(&m)
		b.ReportMetric(float64(m.TotalAlloc-beforeAlloc)/float64(iterations), "B/op")

		badStructSize := unsafe.Sizeof(samples.BadLayoutStruct{})
		structSize := unsafe.Sizeof(samples.OptimizedLayoutStruct{})
		b.Logf("optimized layout struct size: %d bytes", structSize)
		b.Logf("memory impact: optimized layout uses %.2f MB per 1 million instances", float64(structSize)*1_000_000/1024/1024)

		bytesSaved := badStructSize - structSize
		mbSaved := float64(bytesSaved) * 1_000_000 / 1024 / 1024
		percentSaved := float64(bytesSaved) / float64(badStructSize) * 100

		b.Logf("optimization saves %.2f MB per 1 million instances (%.1f%% reduction)", mbSaved, percentSaved)
		b.Logf("for larger systems: saves %.2f GB per billion instances", mbSaved/1024)
	})
}

func BenchmarkAdvancedCachePerformance(b *testing.B) {
	const arraySize = 10_000

	badLayoutStructs := make([]samples.BadLayoutStruct, arraySize)
	optimizedLayoutStructs := make([]samples.OptimizedLayoutStruct, arraySize)

	for i := 0; i < arraySize; i++ {
		badLayoutStructs[i] = samples.BadLayoutStruct{
			ID:           uint64(i),
			IsActive:     i%2 == 0,
			Name:         "user" + string(rune(i%26+'a')),
			Age:          uint8(i % 100),
			Score:        float64(i * 10),
			LastLogin:    time.Now(),
			IsVerified:   i%5 == 0,
			Balance:      float32(i * 100),
			Email:        "user" + string(rune(i%26+'a')) + "@example.com",
			PhoneNumber:  "1234567890",
			IsPremium:    i%3 == 0,
			CreatedAt:    time.Now().AddDate(-1, 0, 0),
			UpdatedAt:    time.Now(),
			Status:       uint16(i % 1000),
			IsBlocked:    i%7 == 0,
			LoginCount:   uint32(i),
			LastIP:       "192.168.1." + string(rune(i%255+'0')),
			IsSuspended:  i%11 == 0,
			SessionToken: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
			IsDeleted:    i%13 == 0,
			ProfilePic:   []byte("profile data"),
			IsLocked:     i%17 == 0,
			FailedLogins: uint8(i % 5),
			IsArchived:   i%19 == 0,
			LastSync:     time.Now(),
			IsHidden:     i%23 == 0,
			Version:      uint64(i),
			IsPinned:     i%29 == 0,
			LastBackup:   time.Now(),
			IsStarred:    i%31 == 0,
			CustomFlags:  uint32(i),
			IsFavorited:  i%37 == 0,
			LastModified: time.Now(),
			IsProtected:  i%41 == 0,
			AccessLevel:  uint8(i % 10),
			IsEncrypted:  i%43 == 0,
			LastAccessed: time.Now(),
			IsCompressed: i%47 == 0,
			FileSize:     int64(i * 1000),
			IsIndexed:    i%53 == 0,
		}

		optimizedLayoutStructs[i] = samples.OptimizedLayoutStruct{
			ID:           badLayoutStructs[i].ID,
			IsActive:     badLayoutStructs[i].IsActive,
			Name:         badLayoutStructs[i].Name,
			Age:          badLayoutStructs[i].Age,
			Score:        badLayoutStructs[i].Score,
			LastLogin:    badLayoutStructs[i].LastLogin,
			IsVerified:   badLayoutStructs[i].IsVerified,
			Balance:      badLayoutStructs[i].Balance,
			Email:        badLayoutStructs[i].Email,
			PhoneNumber:  badLayoutStructs[i].PhoneNumber,
			IsPremium:    badLayoutStructs[i].IsPremium,
			CreatedAt:    badLayoutStructs[i].CreatedAt,
			UpdatedAt:    badLayoutStructs[i].UpdatedAt,
			Status:       badLayoutStructs[i].Status,
			IsBlocked:    badLayoutStructs[i].IsBlocked,
			LoginCount:   badLayoutStructs[i].LoginCount,
			LastIP:       badLayoutStructs[i].LastIP,
			IsSuspended:  badLayoutStructs[i].IsSuspended,
			SessionToken: badLayoutStructs[i].SessionToken,
			IsDeleted:    badLayoutStructs[i].IsDeleted,
			ProfilePic:   badLayoutStructs[i].ProfilePic,
			IsLocked:     badLayoutStructs[i].IsLocked,
			FailedLogins: badLayoutStructs[i].FailedLogins,
			IsArchived:   badLayoutStructs[i].IsArchived,
			LastSync:     badLayoutStructs[i].LastSync,
			IsHidden:     badLayoutStructs[i].IsHidden,
			Version:      badLayoutStructs[i].Version,
			IsPinned:     badLayoutStructs[i].IsPinned,
			LastBackup:   badLayoutStructs[i].LastBackup,
			IsStarred:    badLayoutStructs[i].IsStarred,
			CustomFlags:  badLayoutStructs[i].CustomFlags,
			IsFavorited:  badLayoutStructs[i].IsFavorited,
			LastModified: badLayoutStructs[i].LastModified,
			IsProtected:  badLayoutStructs[i].IsProtected,
			AccessLevel:  badLayoutStructs[i].AccessLevel,
			IsEncrypted:  badLayoutStructs[i].IsEncrypted,
			LastAccessed: badLayoutStructs[i].LastAccessed,
			IsCompressed: badLayoutStructs[i].IsCompressed,
			FileSize:     badLayoutStructs[i].FileSize,
			IsIndexed:    badLayoutStructs[i].IsIndexed,
		}
	}

	badSize := unsafe.Sizeof(samples.BadLayoutStruct{})
	optSize := unsafe.Sizeof(samples.OptimizedLayoutStruct{})
	bytesSaved := badSize - optSize
	mbSavedPerMillion := float64(bytesSaved) * 1_000_000 / 1024 / 1024
	percentReduction := float64(bytesSaved) / float64(badSize) * 100

	b.Logf("struct sizes - bad: %d bytes, optimized: %d bytes", badSize, optSize)
	b.Logf("memory savings: %d bytes per struct (%.1f%% reduction)", bytesSaved, percentReduction)
	b.Logf("scale impact: %.2f MB saved per 1 million instances", mbSavedPerMillion)
	b.Logf("enterprise scale: %.2f GB saved per billion instances", mbSavedPerMillion/1024)

	b.Run("BadLayout_BooleanAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			count := 0
			for i := 0; i < arraySize; i++ {
				if badLayoutStructs[i].IsActive && badLayoutStructs[i].IsPremium {
					count++
				}
				if badLayoutStructs[i].IsVerified && badLayoutStructs[i].IsBlocked {
					count++
				}
			}
			if count < 0 {
				b.Fatalf("unexpected count")
			}
		}
	})

	b.Run("OptimizedLayout_BooleanAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			count := 0
			for i := 0; i < arraySize; i++ {
				if optimizedLayoutStructs[i].IsActive && optimizedLayoutStructs[i].IsPremium {
					count++
				}
				if optimizedLayoutStructs[i].IsVerified && optimizedLayoutStructs[i].IsBlocked {
					count++
				}
			}
			if count < 0 {
				b.Fatalf("unexpected count")
			}
		}
	})

	b.Run("BadLayout_NumericAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			sum := int64(0)
			for i := 0; i < arraySize; i++ {
				sum += int64(badLayoutStructs[i].ID)
				sum += int64(badLayoutStructs[i].LoginCount)
				sum += int64(badLayoutStructs[i].Age)
			}
			if sum < 0 {
				b.Fatalf("unexpected sum")
			}
		}
	})

	b.Run("OptimizedLayout_NumericAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			sum := int64(0)
			for i := 0; i < arraySize; i++ {
				sum += int64(optimizedLayoutStructs[i].ID)
				sum += int64(optimizedLayoutStructs[i].LoginCount)
				sum += int64(optimizedLayoutStructs[i].Age)
			}
			if sum < 0 {
				b.Fatalf("unexpected sum")
			}
		}
	})

	b.Run("BadLayout_StringAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			totalLen := 0
			for i := 0; i < arraySize; i++ {
				totalLen += len(badLayoutStructs[i].Name)
				totalLen += len(badLayoutStructs[i].Email)
				totalLen += len(badLayoutStructs[i].LastIP)
			}
			if totalLen < 0 {
				b.Fatalf("unexpected length")
			}
		}
	})

	b.Run("OptimizedLayout_StringAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			totalLen := 0
			for i := 0; i < arraySize; i++ {
				totalLen += len(optimizedLayoutStructs[i].Name)
				totalLen += len(optimizedLayoutStructs[i].Email)
				totalLen += len(optimizedLayoutStructs[i].LastIP)
			}
			if totalLen < 0 {
				b.Fatalf("unexpected length")
			}
		}
	})

	b.Run("BadLayout_ByteSliceAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			totalLen := 0
			for i := 0; i < arraySize; i++ {
				totalLen += len(badLayoutStructs[i].ProfilePic)
				for j := 0; j < min(len(badLayoutStructs[i].ProfilePic), 3); j++ {
					totalLen += int(badLayoutStructs[i].ProfilePic[j])
				}
				for j := 0; j < min(len(badLayoutStructs[i].SessionToken), 3); j++ {
					totalLen += int(badLayoutStructs[i].SessionToken[j])
				}
			}
			if totalLen < 0 {
				b.Fatalf("unexpected total")
			}
		}
	})

	b.Run("OptimizedLayout_ByteSliceAccess", func(b *testing.B) {
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			totalLen := 0
			for i := 0; i < arraySize; i++ {
				totalLen += len(optimizedLayoutStructs[i].ProfilePic)
				for j := 0; j < min(len(optimizedLayoutStructs[i].ProfilePic), 3); j++ {
					totalLen += int(optimizedLayoutStructs[i].ProfilePic[j])
				}
				for j := 0; j < min(len(optimizedLayoutStructs[i].SessionToken), 3); j++ {
					totalLen += int(optimizedLayoutStructs[i].SessionToken[j])
				}
			}
			if totalLen < 0 {
				b.Fatalf("unexpected total")
			}
		}
	})
}
