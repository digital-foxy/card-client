package operation

type ID string

const EmptyID ID = ""

type IdGenerator func() ID

type Registry interface {
	RegisterImport(vault string) Handle[*ImportReport]
	RegisterUpdate(vault string) Handle[*UpdateReport]
	RegisterExport(vault string) Handle[*ExportReport]
	RegisterDelete(vault string) Handle[*DeleteReport]
	Complete(id ID) error
	Cancel(id ID) error
	Delete(id ID) error
	ListReports() []UnifiedReport
	ActiveOperations() int
}

type Service interface {
	Cancel(id ID) error
	Delete(id ID) error
	ListReports() []UnifiedReport
	ActiveOperations() int
	HasChanges() bool
}
