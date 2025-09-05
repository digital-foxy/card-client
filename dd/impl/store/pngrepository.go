package store

import (
	"cmp"
	"image"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/r3dpixel/card-client/opts"
	"github.com/r3dpixel/card-client/services/scheme"
	"github.com/r3dpixel/card-parser/png"
	"github.com/r3dpixel/toolkit/filex"
	"github.com/r3dpixel/toolkit/timestamp"
	"github.com/r3dpixel/toolkit/trace"
	"github.com/rs/zerolog/log"
	"github.com/sunshineplan/imgconv"
)

const (
	timestampBase     = 10
	thumbnailFileName = "thumbnail.png"
)

// pngRepository - DB Repository for PNG cards
type pngRepository struct {
	rootDir string
	options opts.PngOptions
}

// newPngRepository - Create a new DB repository for PNG cards
func newPngRepository(rootDir string, opts opts.PngOptions) pngRepository {
	return pngRepository{
		rootDir: rootDir,
		options: opts,
	}
}

// findPng - retrieves the PNG card for the given ResourceID and corresponding with the given timestamp
func (r *pngRepository) findPng(cardID scheme.CardID, timestamp timestamp.Nano) (*png.RawCard, error) {
	// Retrieve the path of the PNG card corresponding with the given timestamp
	cardFilePath := r.getCardPngPath(cardID, timestamp)
	// Read the PNG card from the file path
	return png.FromFile(cardFilePath).Get()
}

// findThumbnail - retrieves the PNG thumbnail for the given ResourceID
func (r *pngRepository) findThumbnail(cardID scheme.CardID) (image.Image, error) {
	// Retrieve the path of the thumbnail for the given ResourceID
	cardFilePath := r.getThumbnailPath(cardID.String())
	// Retrieve the thumbnail image
	return filex.ReadImage(cardFilePath)
}

// findAllPngsForID - returns a list with all the timestamps (sorted descending from newest to oldest) for the given ResourceID
// NOTE: timestamp = PNG card file name
func (r *pngRepository) findAllPngsForID(cardID scheme.CardID) []timestamp.Nano {
	// Create a timestamps slice (timestamp = PNG card file name)
	timestamps := make([]timestamp.Nano, 0)

	// Retrieve the root path of the given ResourceID
	cardRootPath := r.getCardRootPath(cardID)
	// Retrieve directory files
	files, err := os.ReadDir(cardRootPath)
	// If the retrieving failed, return an empty slice (log error)
	if err != nil {
		log.Error().Err(err).
			Str("cardID", cardID.String()).
			Str(trace.PATH, cardRootPath).
			Str(trace.ACTIVITY, "find-cards").
			Msg("Failed to read card directory")
		return timestamps
	}
	// If there are no files return empty slice
	if len(files) == 0 {
		return timestamps
	}

	// Parse files
	for _, currentFile := range files {
		// Exclude thumbnail PNG file
		if currentFile.Name() == thumbnailFileName {
			continue
		}

		// Parse the file name into a timestamp version
		version := r.timestampToInt(currentFile.Name())

		// If the version is valid added it to the slice (otherwise remove it)
		if version >= 0 {
			timestamps = append(timestamps, version)
		} else {
			_ = os.Remove(filepath.Join(cardRootPath, currentFile.Name()))
		}
	}

	// Sort timestamps descending
	slices.SortFunc(timestamps, func(a, b timestamp.Nano) int {
		return cmp.Compare(b, a)
	})

	// Return timestamps slice
	return timestamps
}

// savePng - save the raw card bytes to the file system as image/png
func (r *pngRepository) savePng(cardID scheme.CardID, timestamp timestamp.Nano, characterCard *png.CharacterCard) error {
	// Retrieve directory for saving the PNG card
	cardRootPath := r.getCardRootPath(cardID)
	// Retrieve the path of the PNG card corresponding with the given timestamp
	cardPngPath := r.getCardPngPath(cardID, timestamp)
	// Retrieve the path of the thumbnail PNG
	thumbnailFilePath := r.getThumbnailPath(cardID.String())

	// Ensure the directory structure is created
	if err := os.MkdirAll(cardRootPath, filex.DirectoryPermission); err != nil {
		// If the creation of the directory structure fails, return false
		return trace.Err().
			Wrap(err).
			Field("cardID", cardID.String()).
			Field("version", timestamp).
			Field(trace.PATH, cardRootPath).
			Field(trace.ACTIVITY, "save-card").
			Msg("PNG - Failed to create directory for card")
	}

	// Create the thumbnail
	thumbnail, err := characterCard.Thumbnail(r.options.ThumbnailSize)
	// If thumbnail creation failed, return false
	if err != nil {
		return err
	}
	// Save the thumbnail
	if err = filex.WriteImage(thumbnailFilePath, thumbnail, imgconv.PNG); err != nil {
		// If the saving of the thumbnail fails, return error
		return err
	}

	rawCard, err := characterCard.Encode()
	if err != nil {
		return err
	}

	// Save the PNG card at the specified path
	if err = rawCard.ToFile(cardPngPath); err != nil {
		// If the saving of the card failed, return error
		return err
	}
	// Remove the oldest PNG cards in case the number exceeds the allowed number of backup cards
	if err = r.removeOldestPngs(cardID); err != nil {
		// If the removal fails a log message (there is no reason to fail operation)
		log.Error().Err(err).
			Str("cardID", cardID.String()).
			Str(trace.PATH, cardRootPath).
			Str(trace.ACTIVITY, "save-card").
			Msg("Failed to remove old versions of card")
	}

	// Operation succeeded
	return nil
}

// deletePng - remove the PNG card of the given ResourceID, corresponding with the given timestamp
func (r *pngRepository) deletePng(cardID scheme.CardID, timestamp timestamp.Nano) error {
	// Retrieve the PNG card path of the given ResourceID, corresponding with the given timestamp
	cardFilePath := r.getCardPngPath(cardID, timestamp)

	// Remove the PNG card
	if err := os.Remove(cardFilePath); err != nil {
		// If the removal fails, return false (log error)
		return trace.Err().
			Wrap(err).
			Field("cardID", cardID.String()).
			Field("version", timestamp).
			Field(trace.PATH, cardFilePath).
			Field(trace.ACTIVITY, "delete-card").
			Msg("Failed to delete PNG card")
	}

	// Operation succeeded
	return nil
}

// deleteAllPngs - delete all PNG data of the given ResourceID (including thumbnail)
func (r *pngRepository) deleteAllPngs(cardID scheme.CardID) error {
	// Retrieve the root path of the given ResourceID
	cardRootPath := r.getCardRootPath(cardID)

	// Remove the entire directory
	if err := os.RemoveAll(cardRootPath); err != nil {
		// If the removal fails return false (log error)
		return trace.Err().
			Wrap(err).
			Field("cardID", cardID.String()).
			Field(trace.PATH, cardRootPath).
			Field(trace.ACTIVITY, "delete-cards").
			Msg("Failed to delete PNG cards")
	}

	// Operation succeeded
	return nil
}

// removeOldestPngs - removes the oldest PNG cards, enforcing the total number of PNG cards
// for the given ResourceID is not above maxVersions
func (r *pngRepository) removeOldestPngs(cardID scheme.CardID) error {
	// Retrieve all PNG card files names, in timestamp format
	versions := r.findAllPngsForID(cardID)
	// If the number of PNG cards is lower or equal to max, allowed versions return NO-OP success
	if len(versions) <= r.options.MaxVersions {
		return nil
	}
	var tr error
	// Remove the oldest PNG cards, above the maximum allowed versions
	// The PNG cards are already sorted in descending order, which means the last PNG cards in the slice must be removed
	for _, version := range versions[r.options.MaxVersions:] {
		// Remove the PNG card
		if err := r.deletePng(cardID, version); err != nil {

		}
	}

	// Return a success flag
	return tr
}

// getCardRootPath - retrieves the PNG card root path (the directory containing thumbnail and all versions)
func (r *pngRepository) getCardRootPath(cardID scheme.CardID) string {
	return filepath.Join(r.rootDir, cardID.String())
}

// getCardPngPath - retrieves the PNG card file path for the given timestamp
func (r *pngRepository) getCardPngPath(cardID scheme.CardID, timestamp timestamp.Nano) string {
	return filepath.Join(r.rootDir, cardID.String(), r.timestampToString(timestamp))
}

// getThumbnailPath - retrieves the thumbnail file path for the given root path
func (r *pngRepository) getThumbnailPath(cardID string) string {
	return filepath.Join(r.rootDir, cardID, thumbnailFileName)
}

// timestampToString - converts int64 timestamp to string
func (r *pngRepository) timestampToString(timestamp timestamp.Nano) string {
	return strconv.FormatInt(int64(timestamp), timestampBase)
}

// timestampToInt - converts string timestamp to int64
func (r *pngRepository) timestampToInt(t string) timestamp.Nano {
	// Convert string to int64
	if intTimestamp, err := strconv.ParseInt(t, timestampBase, 64); err == nil {
		return timestamp.Nano(intTimestamp)
	}

	// If conversion fails, return -1 (invalid string)
	return -1
}
