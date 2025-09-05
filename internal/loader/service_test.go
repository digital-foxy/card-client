package loader

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/r3dpixel/card-client/services/filter"

	"github.com/r3dpixel/card-client/internal/ent"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-client/services/store"
	"github.com/r3dpixel/card-client/services/vault"
	"github.com/r3dpixel/card-fetcher/models"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockVaultService struct {
	mock.Mock
}

func (m *MockVaultService) GetVault(name string) (vault.Vault, bool) {
	args := m.Called(name)
	return args.Get(0).(vault.Vault), args.Bool(1)
}
func (m *MockVaultService) VaultCount() int                              { return 0 }
func (m *MockVaultService) GetVaults() []vault.Vault                     { return nil }
func (m *MockVaultService) GetVaultNames() []string                      { return nil }
func (m *MockVaultService) CreateVault(name string) (vault.Vault, error) { return vault.Vault{}, nil }
func (m *MockVaultService) DeleteVault(name string) error                { return nil }

type MockStoreService struct {
	mock.Mock
}

func (m *MockStoreService) VaultName() string {
	args := m.Called()
	return args.Get(0).(string)
}

func (m *MockStoreService) Count(ctx context.Context) int {
	args := m.Called(ctx)
	return args.Get(0).(int)
}

func (m *MockStoreService) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStoreService) FindPagedIDs(ctx context.Context, filter filter.SearchFilter, offset int, limit int) []scheme.CardID {
	args := m.Called(ctx, filter, offset, limit)

	var cards []scheme.CardID
	if arg0 := args.Get(0); arg0 != nil {
		cards = arg0.([]scheme.CardID)
	}

	return cards
}

func (m *MockStoreService) FindCards(ctx context.Context, cardIDs []scheme.CardID) ([]scheme.CardHeader, timestamp.Nano) {
	args := m.Called(ctx, cardIDs)

	var cards []scheme.CardHeader
	if arg0 := args.Get(0); arg0 != nil {
		cards = arg0.([]scheme.CardHeader)
	}

	ts := args.Get(1).(timestamp.Nano)

	return cards, ts
}

func (m *MockStoreService) FindIdExportHeaders(ctx context.Context, cardIDs []scheme.CardID) ([]scheme.IdExportHeader, timestamp.Nano) {
	args := m.Called(ctx, cardIDs)

	var payloads []scheme.IdExportHeader
	if arg0 := args.Get(0); arg0 != nil {
		payloads = arg0.([]scheme.IdExportHeader)
	}

	ts := args.Get(1).(timestamp.Nano)

	return payloads, ts
}

func (m *MockStoreService) FindURLs(ctx context.Context, normalizedURLs []string) []string {
	args := m.Called(ctx, normalizedURLs)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]string)
}

func (m *MockStoreService) FindMiniHeaders(ctx context.Context, cardIDs []scheme.CardID) []scheme.MiniHeader {
	args := m.Called(ctx, cardIDs)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]scheme.MiniHeader)
}
func (m *MockStoreService) FindMiniHeader(ctx context.Context, cardID scheme.CardID) (scheme.MiniHeader, error) {
	args := m.Called(ctx, cardID)
	if args.Get(0) == nil {
		return scheme.MiniHeader{}, nil
	}
	return args.Get(0).(scheme.MiniHeader), args.Error(1)
}

func (m *MockStoreService) FindMiscHeaders(ctx context.Context, cardIDs []scheme.CardID) []scheme.MiscHeader {
	args := m.Called(ctx, cardIDs)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]scheme.MiscHeader)
}

func (m *MockStoreService) FindMiscHeader(ctx context.Context, cardID scheme.CardID) (scheme.MiscHeader, error) {
	args := m.Called(ctx, cardID)
	if args.Get(0) == nil {
		return scheme.MiscHeader{}, nil
	}
	return args.Get(0).(scheme.MiscHeader), args.Error(1)
}

func (m *MockStoreService) GetPngPath(cardID scheme.CardID, version timestamp.Nano) string {
	args := m.Called(cardID, version)
	return args.String(0)
}

func (m *MockStoreService) GetThumbnailPath(cardID string) string {
	args := m.Called(cardID)
	return args.String(0)
}

func (m *MockStoreService) InsertCard(ctx context.Context, metadata *models.Metadata, rawCard *png.CharacterCard, importTime timestamp.Nano, batchOrder int) (*scheme.CardHeader, error) {
	args := m.Called(ctx, metadata, rawCard, importTime, batchOrder)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scheme.CardHeader), args.Error(1)
}

func (m *MockStoreService) UpdateCard(ctx context.Context, cardID scheme.CardID, metadata *models.Metadata, rawCard *png.CharacterCard, checkTime timestamp.Nano) (*scheme.CardHeader, error) {
	args := m.Called(ctx, cardID, metadata, rawCard, checkTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*scheme.CardHeader), args.Error(1)
}

func (m *MockStoreService) UpdateStatus(ctx context.Context, cardID scheme.CardID, checkTime timestamp.Nano, status scheme.UpdateStatus) error {
	args := m.Called(ctx, cardID, checkTime, status)
	return args.Error(0)
}

func (m *MockStoreService) UpdateToLatestExport(ctx context.Context, cardID scheme.CardID, exportTime timestamp.Nano) error {
	args := m.Called(ctx, cardID, exportTime)
	return args.Error(0)
}

func (m *MockStoreService) InsertStandardTags(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockStoreService) ToggleFavorite(ctx context.Context, cardID scheme.CardID) error {
	args := m.Called(ctx, cardID)
	return args.Error(0)
}

func (m *MockStoreService) SetFavorites(ctx context.Context, cardIDs []scheme.CardID, favorite bool) error {
	args := m.Called(ctx, cardIDs, favorite)
	return args.Error(0)
}

func TestNewService(t *testing.T) {
	mockVaultService := new(MockVaultService)
	mockProvider := func(client *ent.Client, vault vault.Vault, pngOpts opts.PngOptions) store.Service {
		return nil
	}

	s := NewService(opts.StoreOptions{}, mockVaultService, mockProvider)
	require.NotNil(t, s)

	assert.Equal(t, mockVaultService, s.vaultService)
}

func TestService_LoadVault(t *testing.T) {
	loaderOpts := opts.StoreOptions{
		DbOptions: opts.DbOptions{
			MaxConnections:  10,
			IdleConnections: 5,
			MaxLifetime:     time.Hour,
		},
	}

	t.Run("fails when vault is not found", func(t *testing.T) {
		mockVaultService := new(MockVaultService)
		mockProvider := func(client *ent.Client, vault vault.Vault, pngOpts opts.PngOptions) store.Service {
			return nil
		}
		s := NewService(loaderOpts, mockVaultService, mockProvider)

		vaultName := "non-existent-vault"
		mockVaultService.On("GetVault", vaultName).Return(vault.Vault{}, false).Once()

		_, err := s.LoadVault(vaultName)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "Vault not found")
		mockVaultService.AssertExpectations(t)
	})

	t.Run("successfully loads a new fresh database", func(t *testing.T) {
		mockVaultService := new(MockVaultService)
		mockStoreService := new(MockStoreService)
		mockProvider := func(client *ent.Client, vault vault.Vault, pngOpts opts.PngOptions) store.Service {
			return mockStoreService
		}
		s := NewService(loaderOpts, mockVaultService, mockProvider)

		vaultName := "new-vault"
		dbPath := t.TempDir() + "/test.db"
		testVault := vault.Vault{
			Name:       vaultName,
			DbFilePath: dbPath,
			CardsDir:   t.TempDir(),
		}

		mockVaultService.On("GetVault", vaultName).Return(testVault, true).Once()
		// FIX: Added mock.Anything to match the context.Context argument.
		mockStoreService.On("InsertStandardTags", mock.Anything).Return(nil).Once()

		storeSvc, err := s.LoadVault(vaultName)

		require.NoError(t, err)
		assert.Equal(t, mockStoreService, storeSvc)

		mockVaultService.AssertExpectations(t)
		mockStoreService.AssertExpectations(t)
	})

	t.Run("successfully loads an existing database", func(t *testing.T) {
		mockVaultService := new(MockVaultService)
		mockStoreService := new(MockStoreService)
		mockProvider := func(client *ent.Client, vault vault.Vault, pngOpts opts.PngOptions) store.Service {
			return mockStoreService
		}
		s := NewService(loaderOpts, mockVaultService, mockProvider)

		vaultName := "existing-vault"
		dbPath := t.TempDir() + "/test.db"

		f, err := os.Create(dbPath)
		require.NoError(t, err)
		f.Close()

		testVault := vault.Vault{
			Name:       vaultName,
			DbFilePath: dbPath,
			CardsDir:   t.TempDir(),
		}

		mockVaultService.On("GetVault", vaultName).Return(testVault, true).Once()

		storeSvc, err := s.LoadVault(vaultName)

		require.NoError(t, err)
		assert.Equal(t, mockStoreService, storeSvc)
		mockVaultService.AssertExpectations(t)
		mockStoreService.AssertNotCalled(t, "InsertStandardTags", mock.Anything)
	})

	t.Run("handles error when inserting standard tags", func(t *testing.T) {
		mockVaultService := new(MockVaultService)
		mockStoreService := new(MockStoreService)
		mockProvider := func(client *ent.Client, vault vault.Vault, pngOpts opts.PngOptions) store.Service {
			return mockStoreService
		}
		s := NewService(loaderOpts, mockVaultService, mockProvider)

		vaultName := "tags-fail-vault"
		dbPath := t.TempDir() + "/test.db"
		testVault := vault.Vault{Name: vaultName, DbFilePath: dbPath}

		expectedError := errors.New("failed to insert tags")
		mockVaultService.On("GetVault", vaultName).Return(testVault, true).Once()
		mockStoreService.On("InsertStandardTags", mock.Anything).Return(expectedError).Once()

		_, err := s.LoadVault(vaultName)

		require.NoError(t, err)
		mockVaultService.AssertExpectations(t)
		mockStoreService.AssertExpectations(t)
	})
}
