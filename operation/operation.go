package operation

import (
	"context"

	"github.com/digital-foxy/card-client/library"
	"github.com/digital-foxy/toolkit/timestamp"
)

// ID uniquely identifies an operation
type ID string

const EmptyID ID = ""

// IdGenerator creates unique operation IDs
type IdGenerator func() ID

type Mutator func(report *Report)

type MutationApplier func(Mutator) error

// Handle provides operation context and metadata
type Handle struct {
	OperationID ID
	Context     context.Context
	TimeStarted timestamp.Nano
}

// Action is the type of operation being performed
type Action string

const (
	Import      Action = "import"
	Update      Action = "update"
	Export      Action = "export"
	ExportVault Action = "export-vault"
	ImportVault Action = "import-vault"
	Delete      Action = "delete"
	Integrity   Action = "integrity"
)

// Status indicates the current state of an operation
type Status string

const (
	Ongoing   Status = "ONGOING"
	Completed Status = "COMPLETED"
	Cancelled Status = "CANCELLED"
)

// UnifiedReport combines operation details and progress
type UnifiedReport struct {
	Details Details
	Report  Report
}

// Details describes an operation's metadata
type Details struct {
	ID          ID
	Action      Action
	Status      Status
	TimeStarted timestamp.Nano
	TimeEnded   timestamp.Nano
	VaultName   library.VaultName
	Disposable  bool
}

// Report tracks operation progress and results
type Report struct {
	Progress    int
	Total       int
	NoSuccesses int
	NoFailures  int

	NoValidURLs   int
	NoInvalidURLs int
	NoDuplicates  int
	NoUnchanges   int
	NoFixes       int

	AuxData []string
}

// Registry manages operation lifecycle
type Registry interface {
	NewOperation(vault library.VaultName, action Action) Handle
	MutateReport(opID ID, mutator func(report *Report)) error
	MarkTerminated(opID ID) error
	Service
}

// Service provides operation queries and cancellation
type Service interface {
	Cancel(opID ID) error
	ListReports() []UnifiedReport
	ActiveOperations() int
	HasChanges() bool
}
