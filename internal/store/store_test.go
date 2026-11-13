package store

import (
	"context"
	"github.com/keynertyc/kipuray/internal/casefile"
	"testing"
	"time"
)

func TestPersistAndRevoke(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := casefile.New("http://localhost:3000")
	if err := s.Put(ctx, c); err != nil {
		t.Fatal(err)
	}
	token, err := s.Share(ctx, c.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Shared(ctx, token); err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("persistence: %v %v", items, err)
	}
	if err := s.Revoke(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Shared(ctx, token); err == nil {
		t.Fatal("revoked link still works")
	}
	expired, _ := s.Share(ctx, c.ID, time.Now().Add(-time.Hour))
	if _, err := s.Shared(ctx, expired); err == nil {
		t.Fatal("expired link works")
	}
	if err := s.Delete(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, c.ID); err == nil {
		t.Fatal("case still exists")
	}
}
