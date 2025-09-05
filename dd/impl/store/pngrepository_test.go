package store

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/scheme"
	p "github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestPngFile(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	buf := new(bytes.Buffer)
	require.NoError(t, png.Encode(buf, img))
	filePath := filepath.Join(t.TempDir(), "testcard.png")
	require.NoError(t, os.WriteFile(filePath, buf.Bytes(), 0644))
	return filePath
}

func getEditableCard(t *testing.T, path string) *p.CharacterCard {
	t.Helper()
	rawCard, err := p.FromFile(path).Get()
	require.NoError(t, err)
	characterCard, err := rawCard.Decode()
	require.NoError(t, err)
	return characterCard
}

func newTestCardID(t *testing.T) scheme.CardID {
	t.Helper()
	return scheme.CardID(uuid.NewString())
}

func TestPngRepository_New(t *testing.T) {
	repo := newPngRepository("test/dir", opts.PngOptions{MaxVersions: 10, ThumbnailSize: 128})
	assert.Equal(t, "test/dir", repo.rootDir)
	assert.Equal(t, 10, repo.options.MaxVersions)
	assert.Equal(t, 128, repo.options.ThumbnailSize)
}

func TestPngRepository_FindPng(t *testing.T) {
	tempDir := t.TempDir()
	repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5, ThumbnailSize: 128})
	cardID := newTestCardID(t)
	ts := timestamp.Nano(time.Now().UnixMilli())
	pngFile := createTestPngFile(t)
	characterCard := getEditableCard(t, pngFile)
	require.NoError(t, repo.savePng(cardID, ts, characterCard))

	t.Run("successfully finds an existing png", func(t *testing.T) {
		foundCard, err := repo.findPng(cardID, ts)
		require.NoError(t, err)
		require.NotNil(t, foundCard)
		rawCard, err := characterCard.Encode()
		assert.NoError(t, err)
		assert.Equal(t, rawCard.RawCharaData, foundCard.RawCharaData)
	})

	t.Run("returns error for non-existent timestamp", func(t *testing.T) {
		_, err := repo.findPng(cardID, 12345)
		assert.Error(t, err)
	})

	t.Run("returns error for non-existent card id", func(t *testing.T) {
		_, err := repo.findPng(newTestCardID(t), ts)
		assert.Error(t, err)
	})
}

func TestPngRepository_FindThumbnail(t *testing.T) {
	tempDir := t.TempDir()
	repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5, ThumbnailSize: 128})
	cardID := newTestCardID(t)
	pngFile := createTestPngFile(t)
	characterCard := getEditableCard(t, pngFile)
	require.NoError(t, repo.savePng(cardID, 1, characterCard))

	t.Run("successfully finds an existing thumbnail", func(t *testing.T) {
		thumb, err := repo.findThumbnail(cardID)
		require.NoError(t, err)
		assert.NotNil(t, thumb)
		assert.Equal(t, 128, thumb.Bounds().Dx())
	})

	t.Run("returns error for non-existent card id", func(t *testing.T) {
		_, err := repo.findThumbnail(newTestCardID(t))
		assert.Error(t, err)
	})
}

func TestPngRepository_FindAllPngsForID(t *testing.T) {
	t.Run("finds and sorts valid timestamps", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{})
		cardID := newTestCardID(t)
		cardRootPath := repo.getCardRootPath(cardID)
		require.NoError(t, os.MkdirAll(cardRootPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(cardRootPath, "2000"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(cardRootPath, "1000"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(cardRootPath, "3000"), []byte{}, 0644))

		versions := repo.findAllPngsForID(cardID)
		assert.Equal(t, []timestamp.Nano{3000, 2000, 1000}, versions)
	})

	t.Run("ignores thumbnail and cleans up invalid files", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{})
		cardID := newTestCardID(t)
		cardRootPath := repo.getCardRootPath(cardID)
		invalidFilePath := filepath.Join(cardRootPath, "not-a-number")
		require.NoError(t, os.MkdirAll(cardRootPath, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(cardRootPath, "100"), []byte{}, 0644))
		require.NoError(t, os.WriteFile(filepath.Join(cardRootPath, thumbnailFileName), []byte{}, 0644))
		require.NoError(t, os.WriteFile(invalidFilePath, []byte{}, 0644))

		versions := repo.findAllPngsForID(cardID)
		assert.Equal(t, []timestamp.Nano{100}, versions)
		_, err := os.Stat(invalidFilePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("returns empty for non-existent or unreadable directory", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{})
		cardID := newTestCardID(t)
		versions := repo.findAllPngsForID(cardID)
		assert.NotNil(t, versions)
		assert.Empty(t, versions)
		cardRootPath := repo.getCardRootPath(cardID)
		require.NoError(t, os.MkdirAll(cardRootPath, 0000))
		versions = repo.findAllPngsForID(cardID)
		assert.NotNil(t, versions)
		assert.Empty(t, versions)
		require.NoError(t, os.Chmod(cardRootPath, 0755))
	})
}

func TestPngRepository_SavePng(t *testing.T) {
	pngFile := createTestPngFile(t)
	characterCard := getEditableCard(t, pngFile)

	t.Run("successfully saves card and thumbnail", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5, ThumbnailSize: 64})
		cardID := newTestCardID(t)
		ts := timestamp.Nano(12345)

		err := repo.savePng(cardID, ts, characterCard)
		require.NoError(t, err)

		_, err = os.Stat(repo.getCardPngPath(cardID, ts))
		assert.NoError(t, err, "Card png file should exist")

		_, err = os.Stat(repo.getThumbnailPath(cardID.String()))
		assert.NoError(t, err, "Thumbnail file should exist")
	})

	t.Run("fails to save in read-only directory", func(t *testing.T) {
		tempDir := t.TempDir()
		require.NoError(t, os.Chmod(tempDir, 0555))
		defer func() { require.NoError(t, os.Chmod(tempDir, 0755)) }()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5})
		cardID := newTestCardID(t)
		err := repo.savePng(cardID, 1, characterCard)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrPermission)
	})

	t.Run("triggers cleanup logic correctly", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 2, ThumbnailSize: 64})
		cardID := newTestCardID(t)

		require.NoError(t, repo.savePng(cardID, 100, characterCard))
		require.NoError(t, repo.savePng(cardID, 200, characterCard))
		require.NoError(t, repo.savePng(cardID, 300, characterCard))

		versions := repo.findAllPngsForID(cardID)
		assert.Equal(t, []timestamp.Nano{300, 200}, versions)
		_, err := os.Stat(repo.getCardPngPath(cardID, 100))
		assert.True(t, os.IsNotExist(err))
	})
}

func TestPngRepository_DeletePng(t *testing.T) {
	tempDir := t.TempDir()
	repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5, ThumbnailSize: 128})
	cardID := newTestCardID(t)
	pngFile := createTestPngFile(t)
	characterCard := getEditableCard(t, pngFile)
	require.NoError(t, repo.savePng(cardID, 100, characterCard))
	require.NoError(t, repo.savePng(cardID, 200, characterCard))

	t.Run("successfully deletes one version", func(t *testing.T) {
		err := repo.deletePng(cardID, 100)
		require.NoError(t, err)
		versions := repo.findAllPngsForID(cardID)
		assert.Equal(t, []timestamp.Nano{200}, versions)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		err := repo.deletePng(cardID, 999)
		require.Error(t, err)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})
}

func TestPngRepository_DeleteAllPngs(t *testing.T) {
	tempDir := t.TempDir()
	repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5, ThumbnailSize: 128})
	cardID := newTestCardID(t)
	pngFile := createTestPngFile(t)
	characterCard := getEditableCard(t, pngFile)
	require.NoError(t, repo.savePng(cardID, 100, characterCard))
	require.NoError(t, repo.savePng(cardID, 200, characterCard))

	t.Run("deletes entire directory for existing card", func(t *testing.T) {
		err := repo.deleteAllPngs(cardID)
		require.NoError(t, err)
		_, err = os.Stat(repo.getCardRootPath(cardID))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("is no-op for non-existent card", func(t *testing.T) {
		err := repo.deleteAllPngs(newTestCardID(t))
		assert.NoError(t, err)
	})
}

func TestPngRepository_RemoveOldestPngs(t *testing.T) {
	pngFile := createTestPngFile(t)
	characterCard := getEditableCard(t, pngFile)

	t.Run("does nothing when version count is within limit", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 5, ThumbnailSize: 128})
		cardID := newTestCardID(t)
		require.NoError(t, repo.savePng(cardID, 100, characterCard))
		require.NoError(t, repo.savePng(cardID, 200, characterCard))

		err := repo.removeOldestPngs(cardID)
		require.NoError(t, err)
		versions := repo.findAllPngsForID(cardID)
		assert.Len(t, versions, 2)
	})

	t.Run("removes oldest versions when limit is exceeded", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 3, ThumbnailSize: 128})
		cardID := newTestCardID(t)
		require.NoError(t, repo.savePng(cardID, 100, characterCard))
		require.NoError(t, repo.savePng(cardID, 200, characterCard))
		require.NoError(t, repo.savePng(cardID, 300, characterCard))
		require.NoError(t, repo.savePng(cardID, 400, characterCard))

		err := repo.removeOldestPngs(cardID)
		require.NoError(t, err)
		versions := repo.findAllPngsForID(cardID)
		assert.Equal(t, []timestamp.Nano{400, 300, 200}, versions)
		_, err = os.Stat(repo.getCardPngPath(cardID, 100))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("removes all but one when max versions is 1", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 1, ThumbnailSize: 128})
		cardID := newTestCardID(t)
		require.NoError(t, repo.savePng(cardID, 100, characterCard))
		require.NoError(t, repo.savePng(cardID, 200, characterCard))

		err := repo.removeOldestPngs(cardID)
		require.NoError(t, err)
		versions := repo.findAllPngsForID(cardID)
		assert.Equal(t, []timestamp.Nano{200}, versions)
	})

	t.Run("removes all when max versions is 0", func(t *testing.T) {
		tempDir := t.TempDir()
		repo := newPngRepository(tempDir, opts.PngOptions{MaxVersions: 0, ThumbnailSize: 128})
		cardID := newTestCardID(t)
		require.NoError(t, repo.savePng(cardID, 100, characterCard))

		err := repo.removeOldestPngs(cardID)
		require.NoError(t, err)
		versions := repo.findAllPngsForID(cardID)
		assert.Empty(t, versions)
	})
}

func TestPngRepository_TimestampConversion(t *testing.T) {
	repo := pngRepository{}
	tsNano := timestamp.Nano(123456789)
	tsStr := "123456789"
	assert.Equal(t, tsStr, repo.timestampToString(tsNano))
	assert.Equal(t, tsNano, repo.timestampToInt(tsStr))
	assert.Equal(t, timestamp.Nano(-1), repo.timestampToInt("not-a-number"))
}
