package casefile

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
)

type manifest struct {
	Version int    `json:"version"`
	SHA256  string `json:"sha256"`
}

func Write(w io.Writer, c *Case) error {
	if err := c.Validate(); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if len(data) > MaxBytes {
		return errors.New("case exceeds 64 MiB limit")
	}
	sum := sha256.Sum256(data)
	m, _ := json.Marshal(manifest{Version, hex.EncodeToString(sum[:])})
	z := zip.NewWriter(w)
	for _, entry := range []struct {
		name string
		data []byte
	}{{"manifest.json", m}, {"case.json", data}} {
		f, err := z.Create(entry.name)
		if err != nil {
			return err
		}
		if _, err = f.Write(entry.data); err != nil {
			return err
		}
	}
	return z.Close()
}
func Read(r io.Reader) (*Case, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, errors.New("archive exceeds 64 MiB limit")
	}
	z, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	if len(z.File) != 2 {
		return nil, errors.New("archive must contain manifest.json and case.json only")
	}
	entries := map[string][]byte{}
	var total uint64
	for _, f := range z.File {
		if (f.Name != "manifest.json" && f.Name != "case.json") || entries[f.Name] != nil {
			return nil, errors.New("unexpected or duplicate archive entry")
		}
		total += f.UncompressedSize64
		if total > MaxBytes {
			return nil, errors.New("expanded archive exceeds limit")
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, MaxBytes+1))
		rc.Close()
		if err != nil {
			return nil, err
		}
		if len(b) > MaxBytes {
			return nil, errors.New("entry exceeds limit")
		}
		entries[f.Name] = b
	}
	var m manifest
	if err := json.Unmarshal(entries["manifest.json"], &m); err != nil {
		return nil, err
	}
	sum := sha256.Sum256(entries["case.json"])
	if m.Version != Version || m.SHA256 != hex.EncodeToString(sum[:]) {
		return nil, errors.New("archive version or integrity check failed")
	}
	var c Case
	if err := json.Unmarshal(entries["case.json"], &c); err != nil {
		return nil, err
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}
