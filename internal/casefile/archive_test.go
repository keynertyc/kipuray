package casefile

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestPortableCaseIntegrity(t *testing.T) {
	c := New("http://localhost:3000")
	c.Title = "Coupon regression"
	var b bytes.Buffer
	if err := Write(&b, c); err != nil {
		t.Fatal(err)
	}
	got, err := Read(bytes.NewReader(b.Bytes()))
	if err != nil || got.ID != c.ID {
		t.Fatalf("roundtrip: %v", err)
	}
	var bad bytes.Buffer
	z := zip.NewWriter(&bad)
	f, _ := z.Create("manifest.json")
	f.Write([]byte(`{"version":1,"sha256":"wrong"}`))
	f, _ = z.Create("case.json")
	f.Write([]byte(`{}`))
	z.Close()
	if _, err := Read(&bad); err == nil {
		t.Fatal("accepted tampered archive")
	}
}
func TestRejectUnexpectedArchivePaths(t *testing.T) {
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	for _, p := range []string{"manifest.json", "../../case.json"} {
		f, _ := z.Create(p)
		f.Write([]byte(`{}`))
	}
	z.Close()
	if _, err := Read(&b); err == nil {
		t.Fatal("accepted archive traversal")
	}
}
func TestRequestIdentity(t *testing.T) {
	a, _ := RequestKey("post", "http://localhost:3000/api?z=2&a=1", `{"b":2,"a":1}`)
	b, _ := RequestKey("POST", "http://localhost:3001/api?a=1&z=2", `{ "a": 1, "b": 2 }`)
	if a != b {
		t.Fatalf("equivalent request keys differ: %q, %q", a, b)
	}
}
func TestRequireFunctionalExpectation(t *testing.T) {
	c := New("http://localhost:3000")
	c.Expectations = []Expectation{{Kind: "no-errors"}}
	if c.Runnable() == nil {
		t.Fatal("no functional expectation accepted")
	}
	c.Expectations = append(c.Expectations, Expectation{Kind: "visible", Selector: "#message"})
	if err := c.Runnable(); err != nil {
		t.Fatal(err)
	}
	c.Actions = []Action{{Kind: "input", Redacted: true}}
	if c.Runnable() == nil {
		t.Fatal("masked action accepted")
	}
}
