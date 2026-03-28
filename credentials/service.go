package credentials

import "github.com/digital-foxy/toolkit/cred"

// Label identifies a credential set
type Label = string

const (
	Pygmalion Label = "Pygmalion"
)

var Labels = []Label{Pygmalion}

// Service manages secure credential storage
type Service interface {
	Labels() []Label
	RegisterLabel(label Label)
	GetUsers() map[Label]string
	SetIdentities(payload map[Label]cred.IdentityPayload)
	GetIdentities() map[Label]cred.Identity
	SetIdentity(label Label, payload cred.IdentityPayload)
	GetIdentity(label Label) cred.Identity
	GetReader(label Label) cred.IdentityReader
}

// SecureService exposes limited credential operations
type SecureService interface {
	Labels() []Label
	GetUsers() map[Label]string
	SetIdentities(payload map[Label]cred.IdentityPayload)
}
