package opcache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/elliotchance/orderedmap/v3"
	"github.com/google/uuid"
	"github.com/jaevor/go-nanoid"
	"github.com/r3dpixel/card-client/library"
	"github.com/r3dpixel/card-client/operation"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
)

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
	report  any
	cancel  context.CancelFunc
}

type Registry struct {
	newID      operation.IdGenerator
	mapMutex   sync.RWMutex
	operations orderedmap.OrderedMap[operation.ID, *operationEntry]
	activeOps  atomic.Int32
	dirtyFlag  atomic.Bool
}

func NewRegistry(idGenerator operation.IdGenerator) *Registry {
	return &Registry{
		newID:      idGenerator,
		operations: *orderedmap.NewOrderedMap[operation.ID, *operationEntry](),
	}
}

func register[T any](r *Registry, vault library.VaultName, timeStarted timestamp.Nano, action operation.Action, report T) (operation.ID, context.Context) {
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
		report: report,
		cancel: cancel,
	}

	r.mapMutex.Lock()
	r.operations.Set(id, entry)
	r.mapMutex.Unlock()
	r.dirtyFlag.Store(true)

	return id, ctx

}

func (r *Registry) RegisterImport(vault library.VaultName) operation.Handle[*operation.ImportReport] {
	timeStarted := timestamp.Now[timestamp.Nano]()
	id, ctx := register(r, vault, timeStarted, operation.Import, &operation.ImportReport{})
	return operation.NewHandle(id, ctx, timeStarted, buildApplier[*operation.ImportReport](r, id), buildCompleter(r, id))
}

func (r *Registry) RegisterUpdate(vault library.VaultName) operation.Handle[*operation.UpdateReport] {
	timeStarted := timestamp.Now[timestamp.Nano]()
	id, ctx := register(r, vault, timeStarted, operation.Update, &operation.UpdateReport{})
	return operation.NewHandle(id, ctx, timeStarted, buildApplier[*operation.UpdateReport](r, id), buildCompleter(r, id))
}

func (r *Registry) RegisterExport(vault library.VaultName) operation.Handle[*operation.ExportReport] {
	timeStarted := timestamp.Now[timestamp.Nano]()
	id, ctx := register(r, vault, timeStarted, operation.Export, &operation.ExportReport{})
	return operation.NewHandle(id, ctx, timeStarted, buildApplier[*operation.ExportReport](r, id), buildCompleter(r, id))

}

func (r *Registry) RegisterDelete(vault library.VaultName) operation.Handle[*operation.DeleteReport] {
	timeStarted := timestamp.Now[timestamp.Nano]()
	id, ctx := register(r, vault, timeStarted, operation.Delete, &operation.DeleteReport{})
	return operation.NewHandle(id, ctx, timeStarted, buildApplier[*operation.DeleteReport](r, id), buildCompleter(r, id))
}

func (r *Registry) Complete(opID operation.ID) error {
	return r.end(opID, operation.Completed)
}

func (r *Registry) Cancel(opID operation.ID) error {
	return r.end(opID, operation.Cancelled)
}

func (r *Registry) end(opID operation.ID, finalStatus operation.Status) error {
	r.mapMutex.RLock()
	entry, ok := r.operations.Get(opID)
	r.mapMutex.RUnlock()

	if !ok {
		return fmt.Errorf("Operation %q not found to complete", opID)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.report == nil {
		return fmt.Errorf("Operation report %q is missing", opID)
	}

	if entry.details.Status != operation.Ongoing {
		return fmt.Errorf("Operation %q has already been terminated: %s", opID, entry.details.Status)
	}

	if finalStatus == operation.Cancelled {
		entry.cancel()
	}
	entry.details.TimeEnded = timestamp.Now[timestamp.Nano]()
	entry.details.Status = finalStatus
	r.activeOps.Add(-1)
	r.dirtyFlag.Store(true)

	return nil
}

func (r *Registry) Delete(id operation.ID) error {
	r.mapMutex.Lock()
	defer r.mapMutex.Unlock()
	_, ok := r.operations.Get(id)
	if !ok {
		return fmt.Errorf("Operation %q not found", id)
	}

	r.operations.Delete(id)
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

		unifiedReport := operation.UnifiedReport{Details: entry.details}
		switch specificReport := entry.report.(type) {
		case *operation.ImportReport:
			snapshot := *specificReport
			unifiedReport.Import = &snapshot
		case *operation.UpdateReport:
			snapshot := *specificReport
			unifiedReport.Update = &snapshot
		case *operation.ExportReport:
			snapshot := *specificReport
			unifiedReport.Export = &snapshot
		case *operation.DeleteReport:
			snapshot := *specificReport
			unifiedReport.Delete = &snapshot
		}
		reports = append(reports, unifiedReport)

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

func buildApplier[T interface{}](
	r *Registry,
	id operation.ID,
) operation.MutationApplier[T] {
	return func(mutator operation.Mutator[T]) error {
		r.mapMutex.RLock()
		entry, ok := r.operations.Get(id)
		r.mapMutex.RUnlock()

		if !ok {
			return trace.Err().
				Field(trace.SERVICE, "operation").
				Field(trace.ID, string(id)).
				Msg("Operation with ID not found")
		}

		entry.mu.Lock()
		defer entry.mu.Unlock()

		specificReport, ok := entry.report.(T)
		if !ok {
			return trace.Err().
				Field(trace.SERVICE, "operation").
				Field(trace.ID, string(id)).
				Msg("Operation report type mismatch")
		}

		mutator(specificReport)
		r.dirtyFlag.Store(true)
		return nil
	}
}

func buildCompleter(
	r *Registry,
	id operation.ID,
) func() error {
	return func() error {
		return r.end(id, operation.Completed)
	}
}
