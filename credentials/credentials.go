package credentials

import "github.com/r3dpixel/toolkit/cred"

type Label = string

const (
	Pygmalion Label = "Pygmalion"
)

var Labels = []Label{Pygmalion}

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

type SecureService interface {
	Labels() []Label
	GetUsers() map[Label]string
	SetIdentities(payload map[Label]cred.IdentityPayload)
}
