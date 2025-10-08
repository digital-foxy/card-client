package resource

import (
	"github.com/r3dpixel/toolkit/stringsx"
)

type RID uint64

var EmptyRID RID = RID(0)

type TID string

var EmptyTID TID = TID(stringsx.Empty)

type CID string

var EmptyCID CID = CID(stringsx.Empty)

type SyncStatus string

const (
	SyncFailed    SyncStatus = "FAILED"
	SyncSuccess   SyncStatus = "SUCCESS"
	SyncUnchanged SyncStatus = "UNCHANGED"
	SyncMissing   SyncStatus = "MISSING"
)

func (SyncStatus) Values() []string {
	return []string{
		string(SyncUnchanged),
		string(SyncSuccess),
		string(SyncFailed),
		string(SyncMissing),
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
