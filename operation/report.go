package operation

import (
	"github.com/r3dpixel/card-client/store/resource"
	"github.com/r3dpixel/toolkit/timestamp"
)

type Action string

const (
	Import Action = "import"
	Update Action = "update"
	Export Action = "export"
	Delete Action = "delete"
)

type Status string

const (
	Ongoing   Status = "ONGOING"
	Completed Status = "COMPLETED"
	Cancelled Status = "CANCELLED"
)

type Details struct {
	ID          ID
	Action      Action
	Status      Status
	TimeStarted timestamp.Nano
	TimeEnded   timestamp.Nano
	VaultName   string
	Disposable  bool
}

type Report struct {
	Progress    int
	Total       int
	NoSuccesses int
	NoFailures  int
}

type ImportReport struct {
	Report
	NoValidURLs   int
	NoInvalidURLs int
	NoDuplicates  int
}

type UpdateReport struct {
	Report
	NoUnchanges int
}

type ExportReport struct {
	Report
}

type DeleteReport struct {
	Report
}

type UnifiedReport struct {
	Details Details
	Import  *ImportReport
	Update  *UpdateReport
	Export  *ExportReport
	Delete  *DeleteReport
}

type UpdatePayload struct {
	ResourceID  resource.RID
	OperationID ID
	resource.InfoData
	resource.SyncData
}

type ExportPayload struct {
	OperationID ID
	ResourceID  resource.RID
	resource.ExportData
}
