package opcache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/elliotchance/orderedmap/v3"
	"github.com/google/uuid"
	"github.com/jaevor/go-nanoid"
	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/card-client/operation"
	"github.com/digital-foxy/toolkit/async"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/digital-foxy/toolkit/trace"
)

// DefaultIdGenerator returns a nano ID generator, falling back to UUID
func DefaultIdGenerator() operation.IdGenerator {
	if nanoIdGenerator, err := nanoid.Canonic(); err == nil {
		return func() operation.ID {
			return operation.ID(nanoIdGenerator())
		}
	}
	return func() operation.ID {
		return operation.ID(uuid.NewString())
	}
}

type operationEntry struct {
	mu      sync.Mutex
	details operation.Details
	report  operation.Report
	ctx     context.Context
	cancel  context.CancelFunc
}

// Registry implements operation.Registry with in-memory storage
type Registry struct {
	newID      operation.IdGenerator
	mapMutex   sync.RWMutex
	operations orderedmap.OrderedMap[operation.ID, *operationEntry]
	activeOps  atomic.Int32
	dirtyFlag  atomic.Bool
}

// NewRegistry creates a new operation registry
func NewRegistry(idGenerator operation.IdGenerator) *Registry {
	return &Registry{
		newID:      idGenerator,
		operations: *orderedmap.NewOrderedMap[operation.ID, *operationEntry](),
	}
}

func (r *Registry) NewOperation(vault library.VaultName, action operation.Action) operation.Handle {
	timeStarted := timestamp.NowNano()

	r.activeOps.Add(1)
	id := r.newID()

	ctx, cancel := context.WithCancel(context.Background())

	entry := &operationEntry{
		details: operation.Details{
			ID:          id,
			Action:      action,
			Status:      operation.Ongoing,
			TimeStarted: timeStarted,
			TimeEnded:   0,
			VaultName:   vault,
			Disposable:  false,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	r.mapMutex.Lock()
	r.operations.Set(id, entry)
	r.mapMutex.Unlock()
	r.dirtyFlag.Store(true)

	return operation.Handle{
		OperationID: id,
		Context:     ctx,
		TimeStarted: timeStarted,
	}
}

func (r *Registry) getEntry(opID operation.ID) (*operationEntry, bool) {
	r.mapMutex.RLock()
	defer r.mapMutex.RUnlock()
	return r.operations.Get(opID)
}

func (r *Registry) MarkTerminated(opID operation.ID) error {
	entry, ok := r.getEntry(opID)
	if !ok {
		return fmt.Errorf("operation %q not found", opID)
	}

	status := operation.Completed
	if async.IsCancelled(entry.ctx) {
		status = operation.Cancelled
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.details.Status != operation.Ongoing {
		return fmt.Errorf("operation %q has already been terminated: %s", opID, entry.details.Status)
	}

	entry.details.TimeEnded = timestamp.NowNano()
	entry.details.Status = status
	r.activeOps.Add(-1)
	r.dirtyFlag.Store(true)

	return nil
}

func (r *Registry) Cancel(opID operation.ID) error {
	entry, ok := r.getEntry(opID)
	if !ok {
		return fmt.Errorf("operation %q not found", opID)
	}
	entry.cancel()
	return nil
}

func (r *Registry) MutateReport(opID operation.ID, mutator func(report *operation.Report)) error {
	entry, ok := r.getEntry(opID)
	if !ok {
		return trace.Error().
			Field(trace.SERVICE, "operation").
			Field(trace.ID, string(opID)).
			Msg("Operation with ID not found")
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	mutator(&entry.report)
	r.dirtyFlag.Store(true)
	return nil
}

func (r *Registry) ListReports() []operation.UnifiedReport {
	r.dirtyFlag.Store(false)

	r.mapMutex.RLock()

	toClean := make([]operation.ID, 0)
	reports := make([]operation.UnifiedReport, 0, r.operations.Len())

	for _, entry := range r.operations.AllFromFront() {
		entry.mu.Lock()

		if entry.details.Disposable {
			toClean = append(toClean, entry.details.ID)
			entry.mu.Unlock()
			continue
		}
		if entry.details.Status != operation.Ongoing {
			entry.details.Disposable = true
		}

		reports = append(reports, operation.UnifiedReport{Details: entry.details, Report: entry.report})
		entry.report.AuxData = nil

		entry.mu.Unlock()
	}

	r.mapMutex.RUnlock()

	if len(toClean) > 0 {
		r.mapMutex.Lock()
		for _, id := range toClean {
			r.operations.Delete(id)
		}
		r.mapMutex.Unlock()
	}

	return reports
}

func (r *Registry) ActiveOperations() int {
	return int(r.activeOps.Load())
}

func (r *Registry) HasChanges() bool {
	return r.dirtyFlag.Load()
}
