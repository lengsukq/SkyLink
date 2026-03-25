package store

import (
	"database/sql"
	"strings"
	"testing"
)

func TestDriveEntriesCursor_RoundTrip(t *testing.T) {
	in := &driveEntriesCursor{V: "hello", ID: 123}
	enc := encodeDriveEntriesCursor(in)
	out, err := decodeDriveEntriesCursor(enc)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if out == nil {
		t.Fatalf("decoded cursor is nil")
	}
	if out.ID != in.ID {
		t.Fatalf("id mismatch: got %d want %d", out.ID, in.ID)
	}
	if s, ok := out.V.(string); !ok || s != "hello" {
		t.Fatalf("v mismatch: got %#v", out.V)
	}
}

func TestDriveEntriesCursor_EmptyIsNil(t *testing.T) {
	out, err := decodeDriveEntriesCursor("")
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil cursor, got %#v", out)
	}
}

func TestDriveEntriesCursor_InvalidID(t *testing.T) {
	enc := encodeDriveEntriesCursor(&driveEntriesCursor{V: "x", ID: 0})
	_, err := decodeDriveEntriesCursor(enc)
	if err == nil {
		t.Fatalf("expected error")
	}
	if err != sql.ErrNoRows {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestDriveEntriesCursorWhere_UsesOperatorByOrder(t *testing.T) {
	cur := &driveEntriesCursor{V: "abc", ID: 9}

	whereASC, _ := driveEntriesCursorWhere("ASC", "LOWER(name)", cur)
	if whereASC == "" || !strings.Contains(whereASC, ">") {
		t.Fatalf("expected ASC cursor where uses >, got %q", whereASC)
	}

	whereDESC, _ := driveEntriesCursorWhere("DESC", "LOWER(name)", cur)
	if whereDESC == "" || !strings.Contains(whereDESC, "<") {
		t.Fatalf("expected DESC cursor where uses <, got %q", whereDESC)
	}
}

