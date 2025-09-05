package scheme

import (
	"database/sql/driver"
	"fmt"

	"github.com/r3dpixel/toolkit/stringsx"
)

type CardID string

var EmptyCardID CardID = CardID(stringsx.Empty)

func (i CardID) String() string {
	return string(i)
}

func (i *CardID) Scan(src interface{}) error {
	value, ok := src.(string)
	if !ok {
		return fmt.Errorf("%+v isn't string", src)
	}
	*i = CardID(value)
	return nil
}

func (i CardID) Value() (driver.Value, error) {
	return string(i), nil
}

type UpdateStatus string

const (
	UpdateFailed    UpdateStatus = "FAILED"
	UpdateSuccess   UpdateStatus = "SUCCESS"
	UpdateUnchanged UpdateStatus = "UNCHANGED"
	UpdateMissing   UpdateStatus = "MISSING"
)

func (UpdateStatus) Values() []string {
	return []string{
		string(UpdateUnchanged),
		string(UpdateSuccess),
		string(UpdateFailed),
	}
}

type ImportStatus string

const (
	ImportFailed    ImportStatus = "FAILED"
	ImportSuccess   ImportStatus = "SUCCESS"
	ImportDuplicate ImportStatus = "DUPLICATE"
)
