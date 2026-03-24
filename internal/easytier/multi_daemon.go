package easytier

import (
	"context"
	"fmt"
	"sync"
)

// MultiDaemonManager 负责按 profile ID 管理多个 EasyTier 守护进程实例。
type MultiDaemonManager interface {
	Start(ctx context.Context, profileID string, cfg DaemonConfig) error
	Stop(ctx context.Context, profileID string) error
	Restart(ctx context.Context, profileID string, cfg DaemonConfig) error
	Status(profileID string) DaemonState
	Logs(profileID string) string
	StatusAll() map[string]DaemonState
	StopAll(ctx context.Context) error
}

type multiDaemonManager struct {
	mu       sync.RWMutex
	managers map[string]DaemonManager
}

func NewMultiDaemonManager() MultiDaemonManager {
	return &multiDaemonManager{
		managers: make(map[string]DaemonManager),
	}
}

func (m *multiDaemonManager) Start(ctx context.Context, profileID string, cfg DaemonConfig) error {
	profileID = normalizeProfileID(profileID)
	dm := m.getOrCreate(profileID)
	return dm.Start(ctx, cfg)
}

func (m *multiDaemonManager) Stop(ctx context.Context, profileID string) error {
	profileID = normalizeProfileID(profileID)
	dm := m.getOrCreate(profileID)
	return dm.Stop(ctx)
}

func (m *multiDaemonManager) Restart(ctx context.Context, profileID string, cfg DaemonConfig) error {
	profileID = normalizeProfileID(profileID)
	dm := m.getOrCreate(profileID)
	if err := dm.Stop(ctx); err != nil {
		return err
	}
	return dm.Start(ctx, cfg)
}

func (m *multiDaemonManager) Status(profileID string) DaemonState {
	profileID = normalizeProfileID(profileID)
	dm := m.getOrCreate(profileID)
	return dm.Status()
}

func (m *multiDaemonManager) Logs(profileID string) string {
	profileID = normalizeProfileID(profileID)
	dm := m.getOrCreate(profileID)
	return dm.Logs()
}

func (m *multiDaemonManager) StatusAll() map[string]DaemonState {
	m.mu.RLock()
	ids := make([]string, 0, len(m.managers))
	for id := range m.managers {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	out := make(map[string]DaemonState, len(ids))
	for _, id := range ids {
		out[id] = m.Status(id)
	}
	return out
}

func (m *multiDaemonManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	ids := make([]string, 0, len(m.managers))
	for id := range m.managers {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	var firstErr error
	for _, id := range ids {
		if err := m.Stop(ctx, id); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stop profile %s: %w", id, err)
		}
	}
	return firstErr
}

func (m *multiDaemonManager) getOrCreate(profileID string) DaemonManager {
	m.mu.RLock()
	dm, ok := m.managers[profileID]
	m.mu.RUnlock()
	if ok {
		return dm
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if dm, ok = m.managers[profileID]; ok {
		return dm
	}
	dm = NewDaemonManager()
	m.managers[profileID] = dm
	return dm
}

func normalizeProfileID(profileID string) string {
	if profileID == "" {
		return "default"
	}
	return profileID
}
