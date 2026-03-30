package api

import (
	"strconv"
	"strings"

	"github.com/skylink/skylink/internal/drive"
)

func parseBoolQuery(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func parseIntQuery(v string, def int) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func isSupportedDriveTypeFilter(v string) bool {
	switch v {
	case string(drive.FileTypeImage),
		string(drive.FileTypeVideo),
		string(drive.FileTypeAudio),
		string(drive.FileTypeDocument),
		string(drive.FileTypeArchive),
		string(drive.FileTypeOther):
		return true
	default:
		return false
	}
}

