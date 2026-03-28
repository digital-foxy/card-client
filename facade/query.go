package facade

import (
	"github.com/digital-foxy/card-client/store/resource"
	"github.com/digital-foxy/card-fetcher/router"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/toolkit/timestamp"
)

// queryService provides read-only operations on cards
type queryService struct {
	vault  *vaultManager
	router *router.Router
}

func newQueryService(vault *vaultManager, router *router.Router) *queryService {
	return &queryService{
		vault:  vault,
		router: router,
	}
}

func (s *queryService) CountRecords(filter resource.Filter) (int, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return 0, err
	}
	defer unlock()

	return vault.Catalog.Count(filter)
}

func (s *queryService) FindRIDs(filter resource.Filter) ([]resource.RID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return nil, err
	}
	defer unlock()

	return vault.Catalog.FindPagedRIDs(filter, 0, -1)
}

func (s *queryService) FindPagedIDs(filter resource.Filter, offset int, limit int) ([]resource.RID, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return nil, err
	}
	defer unlock()

	return vault.Catalog.FindPagedRIDs(filter, offset, limit)
}

func (s *queryService) FindRecords(rids ...resource.RID) (resource.Box[resource.Record], error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return resource.Box[resource.Record]{
			Timestamp: timestamp.NowNano(),
		}, err
	}
	defer unlock()

	return vault.Catalog.FindRecords(rids...)
}

func (s *queryService) Sheet(rid resource.RID, version timestamp.Nano) (*character.Sheet, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return nil, err
	}
	defer unlock()

	return vault.Catalog.GetSheet(rid, version)
}

func (s *queryService) ThumbnailBytes(rid resource.RID) ([]byte, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return nil, err
	}
	defer unlock()

	return vault.Catalog.ThumbnailBytes(rid)
}

func (s *queryService) TagNames() ([]string, error) {
	vault, unlock, err := s.vault.beginReadOp()
	if err != nil {
		return nil, err
	}
	defer unlock()

	return vault.Catalog.TagNames()
}

func (s *queryService) GetFilterControls() resource.FieldControls {
	return resource.FieldControls{
		TextFields: []resource.FieldControl{
			{resource.FieldRecordTitle, "Card Name"},
			{resource.FieldCreatorNickname, "Creator"},
			{resource.FieldRecordName, "Name"},
		},
		SortableFields: []resource.FieldControl{
			{resource.FieldRecordUpdateTime, "Update Time"},
			{resource.FieldRecordCreateTime, "Create Time"},
			{resource.FieldRecordImportTime, "Import Time"},
			{resource.FieldRecordSyncTime, "Sync Time"},
		},
		BooleanFields: []resource.FieldControl{
			{resource.FieldRecordFavorite, "Favorite"},
			{resource.FieldRecordIsFork, "Fork"},
			{resource.FieldRecordHasBook, "Has Book"},
		},
		Sources:  s.router.Sources(),
		Statuses: []resource.SyncStatus{resource.SyncSuccess, resource.SyncUnchanged, resource.SyncFailed},
		ContentFields: []resource.FieldControl{
			{resource.FieldContentDescription, "Description"},
			{resource.FieldContentPersonality, "Personality"},
			{resource.FieldContentScenario, "Scenario"},
			{resource.FieldContentFirstMes, "First Message"},
			{resource.FieldContentMesExample, "Message Examples"},
			{resource.FieldContentCreatorNotes, "Creator Notes"},
			{resource.FieldContentSystemPrompt, "System Prompt"},
			{resource.FieldContentPostHistoryInstructions, "Post History Instructions"},
			{resource.FieldContentAlternateGreetings, "Alternate Greetings"},
			{resource.FieldContentDepthPrompt, "Depth Prompt"},
		},
	}
}
