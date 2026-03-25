package service

import (
	"errors"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/skylink/skylink/internal/drive"
	"github.com/skylink/skylink/internal/store"
)

type FilesService struct {
	store *store.Store
}

func NewFilesService(st *store.Store) *FilesService {
	return &FilesService{store: st}
}

func (s *FilesService) Mkdir(acc *store.DriveAccount, userPath string, clientIP string) error {
	if acc == nil {
		return errors.New("account is required")
	}
	full, err := drive.ResolveWithinRoot(acc.RootPath, userPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(full, 0755); err != nil {
		return err
	}

	rel := drive.CleanUserPath(userPath)
	name := filepath.Base(filepath.FromSlash(rel))
	parent := ""
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		parent = rel[:idx]
	}
	_, t := drive.ClassifyFileTypeByName(name, true)
	_ = s.store.UpsertDriveEntry(&store.DriveEntry{
		AccountID:  acc.ID,
		Path:       rel,
		ParentPath: parent,
		Name:       name,
		Ext:        "",
		Type:       string(t),
		IsDir:      true,
		SizeBytes:  0,
		ModifiedAt: 0,
	})
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditMkdir, rel, clientIP)
	return nil
}

func (s *FilesService) Rename(acc *store.DriveAccount, from string, to string, clientIP string) error {
	if acc == nil {
		return errors.New("account is required")
	}
	fromFull, err := drive.ResolveWithinRoot(acc.RootPath, from)
	if err != nil {
		return err
	}
	toFull, err := drive.ResolveWithinRoot(acc.RootPath, to)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(toFull), 0755); err != nil {
		return err
	}
	if err := os.Rename(fromFull, toFull); err != nil {
		return err
	}

	fromRel := drive.CleanUserPath(from)
	toRel := drive.CleanUserPath(to)
	if fromRel != "" {
		_ = s.store.DeleteDriveEntriesByPrefix(acc.ID, fromRel)
	}
	_ = upsertEntryFromFS(s.store, acc, toRel)
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditRename, toRel, clientIP)
	return nil
}

func (s *FilesService) Delete(acc *store.DriveAccount, userPath string, clientIP string) (freedBytes int64, err error) {
	if acc == nil {
		return 0, errors.New("account is required")
	}
	full, err := drive.ResolveWithinRoot(acc.RootPath, userPath)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return 0, err
	}

	var freed int64
	if info.IsDir() {
		freed, _ = computeDirSize(full)
		if err := os.RemoveAll(full); err != nil {
			return 0, err
		}
	} else {
		freed = info.Size()
		if err := os.Remove(full); err != nil {
			return 0, err
		}
	}
	if freed > 0 {
		_ = s.store.AddDriveAccountUsedBytes(acc.ID, -freed)
	}
	rel := drive.CleanUserPath(userPath)
	if rel != "" {
		_ = s.store.DeleteDriveEntriesByPrefix(acc.ID, rel)
	}
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditDelete, rel, clientIP)
	return freed, nil
}

func (s *FilesService) Upload(acc *store.DriveAccount, targetDir string, fh *multipart.FileHeader, clientIP string) (writtenBytes int64, err error) {
	if acc == nil {
		return 0, errors.New("account is required")
	}
	if fh == nil {
		return 0, errors.New("file is required")
	}
	// Refresh account for accurate quota/used check.
	if fresh, err := s.store.GetDriveAccount(acc.ID); err == nil && fresh != nil {
		acc = fresh
	}

	fullDir, err := drive.ResolveWithinRoot(acc.RootPath, targetDir)
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return 0, err
	}

	filename := filepath.Base(fh.Filename)
	if filename == "." || filename == string(filepath.Separator) || strings.TrimSpace(filename) == "" {
		return 0, errors.New("invalid filename")
	}

	if acc.QuotaBytes > 0 && fh.Size > 0 && acc.UsedBytes+fh.Size > acc.QuotaBytes {
		return 0, drive.ErrQuotaExceeded
	}

	dstPath := filepath.Join(fullDir, filename)
	dstPath = filepath.Clean(dstPath)
	if !strings.HasPrefix(strings.ToLower(dstPath), strings.ToLower(filepath.Clean(fullDir))+string(filepath.Separator)) && strings.ToLower(dstPath) != strings.ToLower(fullDir) {
		return 0, errors.New("invalid path")
	}

	src, err := fh.Open()
	if err != nil {
		return 0, err
	}
	defer src.Close()

	tmpPath := dstPath + ".uploading"
	out, err := os.Create(tmpPath)
	if err != nil {
		return 0, err
	}
	written, copyErr := io.Copy(out, src)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return 0, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return 0, closeErr
	}

	if acc.QuotaBytes > 0 && written > 0 && acc.UsedBytes+written > acc.QuotaBytes {
		_ = os.Remove(tmpPath)
		return 0, drive.ErrQuotaExceeded
	}

	if err := os.Rename(tmpPath, dstPath); err != nil {
		_ = os.Remove(tmpPath)
		return 0, err
	}
	if written > 0 {
		_ = s.store.AddDriveAccountUsedBytes(acc.ID, written)
	}
	relDir := drive.CleanUserPath(targetDir)
	rel := filename
	if relDir != "" {
		rel = filepath.ToSlash(filepath.Join(relDir, filename))
	}
	_ = upsertEntryFromFS(s.store, acc, rel)
	_ = s.store.AddDriveAuditLog(acc.ID, store.DriveAuditUpload, rel, clientIP)
	return written, nil
}

func computeDirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func upsertEntryFromFS(st *store.Store, acc *store.DriveAccount, rel string) error {
	if st == nil || acc == nil {
		return nil
	}
	rel = drive.CleanUserPath(rel)
	full, err := drive.ResolveWithinRoot(acc.RootPath, rel)
	if err != nil {
		return err
	}
	info, err := os.Stat(full)
	if err != nil {
		return err
	}
	parent := ""
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		parent = rel[:idx]
	}
	name := info.Name()
	ext, t := drive.ClassifyFileTypeByName(name, info.IsDir())
	entry := &store.DriveEntry{
		AccountID:  acc.ID,
		Path:       rel,
		ParentPath: parent,
		Name:       name,
		Ext:        ext,
		Type:       string(t),
		IsDir:      info.IsDir(),
		SizeBytes:  info.Size(),
		ModifiedAt: info.ModTime().Unix(),
	}
	return st.UpsertDriveEntry(entry)
}

