package resource

import (
	"strconv"
)

// RID is a unique identifier for a resource record
type RID uint64

// String returns the string representation of the RID
func (r RID) String() string {
	return strconv.FormatUint(uint64(r), 10)
}

// EmptyRID represents an empty or invalid resource ID
var EmptyRID RID = RID(0)

// TID is a unique identifier for a tag
type TID string

// EmptyTID represents an empty or invalid tag ID
var EmptyTID TID = TID("")

// CID is a unique identifier for a creator
type CID string

// EmptyCID represents an empty or invalid creator ID
var EmptyCID CID = CID("")

// SyncStatus represents the synchronization status of a record
type SyncStatus string

const (
	SyncFailed    SyncStatus = "FAILED"
	SyncSuccess   SyncStatus = "SUCCESS"
	SyncUnchanged SyncStatus = "UNCHANGED"
	SyncMissing   SyncStatus = "MISSING"
)

// Values returns all possible sync status values
func (SyncStatus) Values() []string {
	return []string{
		string(SyncUnchanged),
		string(SyncSuccess),
		string(SyncFailed),
		string(SyncMissing),
	}
}

// ImportStatus represents the result of an import operation
type ImportStatus string

const (
	ImportFailed    ImportStatus = "FAILED"
	ImportSuccess   ImportStatus = "SUCCESS"
	ImportDuplicate ImportStatus = "DUPLICATE"
)

// Values returns all possible import status values
func (ImportStatus) Values() []string {
	return []string{
		string(ImportFailed),
		string(ImportSuccess),
		string(ImportDuplicate),
	}
}

// RecordIntegrity represents the integrity state of a record
type RecordIntegrity byte

const (
	OK RecordIntegrity = iota
	FIXED
	BROKEN
)
