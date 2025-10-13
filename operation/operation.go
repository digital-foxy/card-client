package operation

import "github.com/r3dpixel/card-client/library"

type ID string

const EmptyID ID = ""

type IdGenerator func() ID

type Registry interface {
	RegisterImport(vault library.VaultName) Handle[*ImportReport]
	RegisterUpdate(vault library.VaultName) Handle[*UpdateReport]
	RegisterExport(vault library.VaultName) Handle[*ExportReport]
	RegisterDelete(vault library.VaultName) Handle[*DeleteReport]
	Complete(opID ID) error
	Cancel(opID ID) error
	Delete(opID ID) error
	ListReports() []UnifiedReport
	ActiveOperations() int
}

type Service interface {
	Cancel(opID ID) error
	Delete(opID ID) error
	ListReports() []UnifiedReport
	ActiveOperations() int
	HasChanges() bool
}
