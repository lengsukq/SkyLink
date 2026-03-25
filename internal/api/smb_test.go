package api

import (
	"errors"
	"testing"
)

func TestMapSMBStoreError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "share_name unique",
			err:  errors.New("UNIQUE constraint failed: smb_mappings.share_name"),
			want: "share_name already exists",
		},
		{
			name: "name unique",
			err:  errors.New("unique constraint failed: smb_mappings.name"),
			want: "name already exists",
		},
		{
			name: "fallback",
			err:  errors.New("other db error"),
			want: "other db error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := mapSMBStoreError(tc.err)
			if got != tc.want {
				t.Fatalf("mapSMBStoreError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestShouldSkipSystemShare(t *testing.T) {
	cases := []struct {
		shareName string
		want      bool
	}{
		{shareName: "ADMIN$", want: true},
		{shareName: "C$", want: true},
		{shareName: "print$", want: true},
		{shareName: "Docs", want: false},
	}
	for _, tc := range cases {
		if got := shouldSkipSystemShare(tc.shareName); got != tc.want {
			t.Fatalf("shouldSkipSystemShare(%q) = %v, want %v", tc.shareName, got, tc.want)
		}
	}
}
