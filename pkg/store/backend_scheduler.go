package store

import (
	"context"
	"log"
	"sync"
	"time"
)

const snapshotDebounce = 800 * time.Millisecond

var (
	snapMu    sync.Mutex
	snapTimer *time.Timer
	snapHub   *Hub
)

// RequestSnapshotSave 在 Hub 有配置 Backend 且为可写实现时，防抖触发一次 Write(ExportState)。
func (h *Hub) RequestSnapshotSave() {
	if h == nil || h.stateBackend == nil {
		return
	}
	if h.stateBackend.Kind() == "memory" {
		return
	}
	snapMu.Lock()
	snapHub = h
	if snapTimer != nil {
		snapTimer.Stop()
	}
	snapTimer = time.AfterFunc(snapshotDebounce, func() {
		snapMu.Lock()
		hub := snapHub
		snapMu.Unlock()
		if hub == nil || hub.stateBackend == nil {
			return
		}
		st := hub.ExportState()
		ctx := context.Background()
		if err := hub.stateBackend.Write(ctx, st); err != nil {
			log.Printf("agentpark: snapshot write (%s): %v", hub.stateBackend.Kind(), err)
		}
	})
	snapMu.Unlock()
}

// FlushSnapshot 同步写盘（退出信号等）。
func (h *Hub) FlushSnapshot() error {
	if h == nil || h.stateBackend == nil {
		return nil
	}
	if h.stateBackend.Kind() == "memory" {
		return nil
	}
	return h.stateBackend.Write(context.Background(), h.ExportState())
}
