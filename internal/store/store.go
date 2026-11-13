// Package store persists cases, execution results, and revocable sharing links.
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/keynertyc/kipuray/internal/casefile"
	_ "modernc.org/sqlite"
)

type Store struct{ db *sql.DB }
type Summary struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Origin    string `json:"origin"`
	CreatedAt string `json:"createdAt"`
	Actions   int    `json:"actions"`
	Fixtures  int    `json:"fixtures"`
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "kipuray.db"))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;
 CREATE TABLE IF NOT EXISTS cases(id TEXT PRIMARY KEY, title TEXT NOT NULL, origin TEXT NOT NULL, created TEXT NOT NULL, body BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS runs(id TEXT PRIMARY KEY, case_id TEXT NOT NULL, created TEXT NOT NULL, body BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS shares(token_hash TEXT PRIMARY KEY, case_id TEXT NOT NULL, expires INTEGER NOT NULL);`)
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Put(ctx context.Context, c *casefile.Case) error {
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if len(b) > casefile.MaxBytes {
		return errors.New("case exceeds 64 MiB")
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO cases VALUES(?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET title=excluded.title,origin=excluded.origin,body=excluded.body`, c.ID, c.Title, c.Origin, c.CreatedAt.Format(time.RFC3339Nano), b)
	return err
}
func (s *Store) Get(ctx context.Context, id string) (*casefile.Case, error) {
	var b []byte
	err := s.db.QueryRowContext(ctx, `SELECT body FROM cases WHERE id=?`, id).Scan(&b)
	if err != nil {
		return nil, err
	}
	var c casefile.Case
	err = json.Unmarshal(b, &c)
	return &c, err
}
func (s *Store) List(ctx context.Context) ([]Summary, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,title,origin,created,json_array_length(body,'$.actions'),json_array_length(body,'$.fixtures') FROM cases ORDER BY created DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		var x Summary
		if err := rows.Scan(&x.ID, &x.Title, &x.Origin, &x.CreatedAt, &x.Actions, &x.Fixtures); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}
func (s *Store) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{`DELETE FROM shares WHERE case_id=?`, `DELETE FROM runs WHERE case_id=?`, `DELETE FROM cases WHERE id=?`} {
		if _, err = tx.ExecContext(ctx, q, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Store) SaveRun(ctx context.Context, caseID string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO runs VALUES(?,?,?,?)`, casefile.NewID(), caseID, time.Now().UTC().Format(time.RFC3339Nano), b)
	return err
}
func (s *Store) Runs(ctx context.Context, id string) ([]json.RawMessage, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT body FROM runs WHERE case_id=? ORDER BY created DESC LIMIT 20`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return nil, err
		}
		out = append(out, json.RawMessage(b))
	}
	return out, rows.Err()
}
func hash(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
func (s *Store) Share(ctx context.Context, id string, expires time.Time) (string, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return "", err
	}
	token := casefile.NewID() + casefile.NewID()
	_, err := s.db.ExecContext(ctx, `INSERT INTO shares VALUES(?,?,?)`, hash(token), id, expires.Unix())
	return token, err
}
func (s *Store) Shared(ctx context.Context, token string) (*casefile.Case, error) {
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT case_id FROM shares WHERE token_hash=? AND expires>?`, hash(token), time.Now().Unix()).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}
func (s *Store) Revoke(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM shares WHERE case_id=?`, id)
	return err
}
