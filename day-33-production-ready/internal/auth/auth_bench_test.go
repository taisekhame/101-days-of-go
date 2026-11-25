package auth

import (
	"testing"
	"time"
)

func newBenchService(b *testing.B) *Service {
	b.Helper()
	svc, err := New(testSecret, time.Hour)
	if err != nil {
		b.Fatal(err)
	}
	return svc
}

func BenchmarkHashPassword(b *testing.B) {
	svc := newBenchService(b)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := svc.HashPassword("mtb-password-123"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCheckPassword(b *testing.B) {
	svc := newBenchService(b)
	hash, err := svc.HashPassword("mtb-password-123")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer() // the setup above should not count

	for i := 0; i < b.N; i++ {
		svc.CheckPassword(hash, "mtb-password-123")
	}
}

func BenchmarkIssueToken(b *testing.B) {
	svc := newBenchService(b)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := svc.IssueToken(42); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseToken(b *testing.B) {
	svc := newBenchService(b)
	token, err := svc.IssueToken(42)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := svc.ParseToken(token); err != nil {
			b.Fatal(err)
		}
	}
}
