package indexer

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/skylink/skylink/internal/drive"
	"github.com/skylink/skylink/internal/store"
)

type Status struct {
	Running      bool   `json:"running"`
	StartedAt    int64  `json:"started_at"`
	FinishedAt   int64  `json:"finished_at"`
	ScannedFiles int64  `json:"scanned_files"`
	ScannedDirs  int64  `json:"scanned_dirs"`
	LastError    string `json:"last_error"`
}

type Manager struct {
	mu      sync.Mutex
	status  map[int64]*Status
	running map[int64]context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{
		status:  map[int64]*Status{},
		running: map[int64]context.CancelFunc{},
	}
}

func (m *Manager) GetStatus(accountID int64) Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.status[accountID]; ok && s != nil {
		return *s
	}
	return Status{}
}

func (m *Manager) StartRebuild(ctx context.Context, st *store.Store, acc *store.DriveAccount) error {
	if st == nil || acc == nil || acc.ID <= 0 {
		return errors.New("invalid args")
	}
	m.mu.Lock()
	if _, ok := m.running[acc.ID]; ok {
		m.mu.Unlock()
		return errors.New("rebuild already running")
	}
	runCtx, cancel := context.WithCancel(ctx)
	m.running[acc.ID] = cancel
	s := &Status{Running: true, StartedAt: time.Now().Unix()}
	m.status[acc.ID] = s
	m.mu.Unlock()

	go func() {
		defer func() {
			m.mu.Lock()
			delete(m.running, acc.ID)
			if ss := m.status[acc.ID]; ss != nil {
				ss.Running = false
				ss.FinishedAt = time.Now().Unix()
			}
			m.mu.Unlock()
		}()

		if err := rebuildAccount(runCtx, st, acc, s); err != nil {
			m.mu.Lock()
			if ss := m.status[acc.ID]; ss != nil {
				ss.LastError = err.Error()
			}
			m.mu.Unlock()
		}
	}()

	return nil
}

func rebuildAccount(ctx context.Context, st *store.Store, acc *store.DriveAccount, status *Status) error {
	// Clear existing entries for account before rebuild.
	if err := st.DeleteDriveEntriesByPrefix(acc.ID, ""); err != nil {
		return err
	}

	root := filepath.Clean(acc.RootPath)
	return filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			rel = ""
		}

		if d.IsDir() {
			if rel != "" {
				status.ScannedDirs++
				name := d.Name()
				parent := parentPath(rel)
				_, t := drive.ClassifyFileTypeByName(name, true)
				_ = st.UpsertDriveEntry(&store.DriveEntry{
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
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		status.ScannedFiles++
		name := d.Name()
		ext, t := drive.ClassifyFileTypeByName(name, false)
		parent := parentPath(rel)
		if err := st.UpsertDriveEntry(&store.DriveEntry{
			AccountID:  acc.ID,
			Path:       rel,
			ParentPath: parent,
			Name:       name,
			Ext:        ext,
			Type:       string(t),
			IsDir:      false,
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().Unix(),
		}); err != nil {
			return err
		}
		return nil
	})
}

func parentPath(p string) string {
	p = strings.TrimSuffix(strings.TrimSpace(p), "/")
	if p == "" {
		return ""
	}
	if idx := strings.LastIndex(p, "/"); idx >= 0 {
		return p[:idx]
	}
	return ""
}
