package facade

import (
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/digital-foxy/card-client/store/catalog"
	"github.com/digital-foxy/card-fetcher/fetcher"
	"github.com/digital-foxy/card-fetcher/models"
	"github.com/digital-foxy/card-fetcher/source"
	"github.com/digital-foxy/card-parser/character"
	"github.com/digital-foxy/card-parser/png"
	"github.com/digital-foxy/toolkit/imagex"
	"github.com/digital-foxy/toolkit/stringsx"
	"github.com/digital-foxy/toolkit/timestamp"
	"github.com/sunshineplan/imgconv"
)

// Signature uniquely identifies an uploaded card
type Signature string

// UploadRequest contains upload data including thumbnail
type UploadRequest struct {
	UploadResponse
	ThumbnailBase64 string
}

// UploadResponse contains confirmed upload data
type UploadResponse struct {
	Signature Signature
	Metadata  *models.Metadata
	Sheet     *character.Sheet
}

// uploadService handles local card uploads
type uploadService struct {
	thumbnailSize int
	catalog       catalog.Service
	storage       map[Signature]*png.CharacterCard
	storageMu     sync.Mutex
}

func newUploadService(thumbnailSize int) *uploadService {
	return &uploadService{
		thumbnailSize: thumbnailSize,
		storage:       make(map[Signature]*png.CharacterCard),
	}
}

func (u *uploadService) LoadLocalCard(path string) (UploadRequest, error) {
	rawCard, err := png.FromFile(path).LastLongest().Get()
	if err != nil {
		return UploadRequest{}, err
	}

	characterCard, err := rawCard.Decode()
	if err != nil {
		return UploadRequest{}, err
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		return UploadRequest{}, err
	}
	stringSignature := uid.String()
	signature := Signature(stringSignature)

	u.storageMu.Lock()
	u.storage[signature] = characterCard
	u.storageMu.Unlock()

	thumbnail, err := characterCard.Thumbnail(u.thumbnailSize)
	if err != nil {
		return UploadRequest{}, err
	}

	thumbnailBytes, err := imagex.ToBytes(thumbnail, imgconv.WEBP)
	if err != nil {
		return UploadRequest{}, err
	}

	title := string(characterCard.Title)
	if stringsx.IsBlank(title) {
		title = string(characterCard.Name)
	}

	tags := make([]models.Tag, 0, len(characterCard.Tags))
	for _, tag := range characterCard.Tags {
		tags = append(tags, models.ResolveTag(tag))
	}
	characterCard.Tags = nil

	metadata := &models.Metadata{
		Source: source.Local,
		CardInfo: models.CardInfo{
			NormalizedURL: stringSignature,
			PlatformID:    stringSignature,
			CharacterID:   stringSignature,
			Name:          string(characterCard.Name),
			Title:         title,
			Tagline:       "",
			CreateTime:    timestamp.NowNano(),
			UpdateTime:    timestamp.NowNano(),
			IsForked:      false,
			Tags:          tags,
		},
		CreatorInfo: models.CreatorInfo{
			Nickname:   string(characterCard.Creator),
			Username:   string(characterCard.Creator),
			PlatformID: stringSignature,
		},
		BookUpdateTime: 0,
		GreetingsCount: len(characterCard.AlternateGreetings),
	}

	return UploadRequest{
		UploadResponse: UploadResponse{
			Signature: signature,
			Metadata:  metadata,
			Sheet:     characterCard.Sheet,
		},
		ThumbnailBase64: base64.StdEncoding.EncodeToString(thumbnailBytes),
	}, nil
}

func (u *uploadService) UnloadLocalCard(signature Signature) {
	u.storageMu.Lock()
	delete(u.storage, signature)
	u.storageMu.Unlock()
}

func (u *uploadService) AcceptLocalCard(response UploadResponse) error {
	u.storageMu.Lock()
	characterCard, ok := u.storage[response.Signature]
	u.storageMu.Unlock()
	if !ok {
		return fmt.Errorf("local card already canceled or saved: %s", response.Signature)
	}

	characterCard.Sheet = response.Sheet

	characterCard.Sheet.Tags = nil
	fetcher.PatchMetadata(response.Metadata)
	fetcher.PatchSheet(characterCard.Sheet, response.Metadata)

	if _, err := u.catalog.SaveCard(response.Metadata, characterCard, timestamp.NowNano()); err != nil {
		return err
	}

	u.storageMu.Lock()
	delete(u.storage, response.Signature)
	u.storageMu.Unlock()

	return nil
}
