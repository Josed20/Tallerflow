package blob

import (
	"context"
	"net/url"
	"testing"
)

func TestPresignedPhotoURLExpiresInFifteenMinutes(t *testing.T) {
	store, err := NewR2(context.Background(), Config{
		Endpoint: "https://account.r2.cloudflarestorage.com",
		Bucket:   "private-photos", AccessKey: "test-access", SecretKey: "test-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	photoURL, err := store.URL(context.Background(), "workshops/one/orders/two/photos/three.jpg")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(photoURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "https" || parsed.Host != "account.r2.cloudflarestorage.com" || parsed.Query().Get("X-Amz-Expires") != "900" || parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatalf("unexpected presigned URL: %s", photoURL)
	}
}

func TestRejectsInsecureR2Endpoint(t *testing.T) {
	_, err := NewR2(context.Background(), Config{Endpoint: "http://localhost:9000", Bucket: "photos", AccessKey: "key", SecretKey: "secret"})
	if err == nil {
		t.Fatal("insecure endpoint accepted")
	}
}
