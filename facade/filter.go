package facade

import (
	"github.com/r3dpixel/card-client/store/resource"
)

func (f *Facade) CountRecords(filter resource.Filter) (int, error) {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return 0, err
	} else {
		defer unlock()
	}
	return f.vault.Catalog.Count(filter)
}

func (f *Facade) FindRIDs(filter resource.Filter) ([]resource.RID, error) {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return nil, err
	} else {
		defer unlock()
	}

	return f.vault.Catalog.FindPagedRIDs(filter, 0, -1)
}
func (f *Facade) FindPagedIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error) {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return nil, err
	} else {
		defer unlock()
	}

	return f.vault.Catalog.FindPagedRIDs(filter, offset, limit)
}

func (f *Facade) FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error) {
	if unlock, err := f.beginReadStoreOp(); err != nil {
		return resource.Box[resource.Record]{}, err
	} else {
		defer unlock()
	}

	return f.vault.Catalog.FindRecords(rids...)
}
