package resource

import "github.com/r3dpixel/toolkit/stringsx"

type RID string

var EmptyRID RID = RID(stringsx.Empty)

type TID string

var EmptyTID TID = TID(stringsx.Empty)

type CID int64

var EmptyCID CID = CID(int64(0))

type SyncStatus string

const (
	UpdateFailed    SyncStatus = "FAILED"
	UpdateSuccess   SyncStatus = "SUCCESS"
	UpdateUnchanged SyncStatus = "UNCHANGED"
	UpdateMissing   SyncStatus = "MISSING"
)

func (SyncStatus) Values() []string {
	return []string{
		string(UpdateUnchanged),
		string(UpdateSuccess),
		string(UpdateFailed),
		string(UpdateMissing),
	}
}

type ImportStatus string

const (
	ImportFailed    ImportStatus = "FAILED"
	ImportSuccess   ImportStatus = "SUCCESS"
	ImportDuplicate ImportStatus = "DUPLICATE"
)

func (ImportStatus) Values() []string {
	return []string{
		string(ImportFailed),
		string(ImportSuccess),
		string(ImportDuplicate),
	}
}
