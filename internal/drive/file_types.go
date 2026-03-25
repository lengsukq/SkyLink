package drive

import (
	"path/filepath"
	"strings"
)

type FileType string

const (
	FileTypeImage    FileType = "image"
	FileTypeVideo    FileType = "video"
	FileTypeAudio    FileType = "audio"
	FileTypeDocument FileType = "document"
	FileTypeArchive  FileType = "archive"
	FileTypeOther    FileType = "other"
	FileTypeFolder   FileType = "folder"
)

func ClassifyFileTypeByName(name string, isDir bool) (ext string, t FileType) {
	if isDir {
		return "", FileTypeFolder
	}
	ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	if ext == "" {
		return "", FileTypeOther
	}
	if imageExts[ext] {
		return ext, FileTypeImage
	}
	if videoExts[ext] {
		return ext, FileTypeVideo
	}
	if audioExts[ext] {
		return ext, FileTypeAudio
	}
	if docExts[ext] {
		return ext, FileTypeDocument
	}
	if archiveExts[ext] {
		return ext, FileTypeArchive
	}
	return ext, FileTypeOther
}

var imageExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true, "bmp": true, "tif": true, "tiff": true, "svg": true, "ico": true, "heic": true, "heif": true,
}

var videoExts = map[string]bool{
	"mp4": true, "mkv": true, "mov": true, "avi": true, "wmv": true, "flv": true, "webm": true, "m4v": true, "ts": true,
}

var audioExts = map[string]bool{
	"mp3": true, "wav": true, "flac": true, "aac": true, "m4a": true, "ogg": true, "opus": true, "wma": true,
}

var docExts = map[string]bool{
	"txt": true, "md": true,
	"pdf": true,
	"doc": true, "docx": true,
	"xls": true, "xlsx": true,
	"ppt": true, "pptx": true,
	"csv": true,
	"json": true, "xml": true, "yaml": true, "yml": true,
	"html": true, "htm": true,
}

var archiveExts = map[string]bool{
	"zip": true, "rar": true, "7z": true, "tar": true, "gz": true, "tgz": true, "bz2": true, "xz": true,
}

