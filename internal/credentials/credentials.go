package credentials

import (
	"slices"
	"sync"

	"github.com/r3dpixel/card-client/services/credentials"
	"github.com/r3dpixel/toolkit/cred"
)

type Service struct {
	mutex    sync.RWMutex
	labels   []credentials.Label
	managers map[credentials.Label]cred.IdentityManager
}

func NewService() *Service {
	s := &Service{
		labels:   slices.Clone(credentials.Labels),
		managers: make(map[credentials.Label]cred.IdentityManager),
	}

	for _, label := range s.labels {
		s.managers[label] = cred.NewManager(string(label), cred.KeyRing)
	}

	return s
}

func (s *Service) Labels() []credentials.Label {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.labels
}

func (s *Service) RegisterLabel(label credentials.Label) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.managers[label]; !ok {
		s.labels = append(s.labels, label)
		s.managers[label] = cred.NewManager(string(label), cred.KeyRing)
	}
}

func (s *Service) SetIdentities(payload map[credentials.Label]cred.IdentityPayload) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for label, p := range payload {
		if manager, ok := s.managers[label]; ok {
			_ = manager.Set(p)
		}
	}
}

func (s *Service) GetIdentities() map[credentials.Label]cred.Identity {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	identities := make(map[credentials.Label]cred.Identity)
	for label, manager := range s.managers {
		if identity, err := manager.Get(); err == nil {
			identities[label] = identity
		}
	}
	return identities
}

func (s *Service) GetUsers() map[credentials.Label]string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	users := make(map[credentials.Label]string)
	for label, manager := range s.managers {
		if identity, err := manager.GetUser(); err == nil {
			users[label] = identity
		}
	}
	return users
}

func (s *Service) SetIdentity(label credentials.Label, payload cred.IdentityPayload) {
	s.mutex.RLock()
	manager, ok := s.managers[label]
	s.mutex.RUnlock()

	if ok {
		_ = manager.Set(payload)
	}
}

func (s *Service) GetIdentity(label credentials.Label) cred.Identity {
	s.mutex.RLock()
	manager, ok := s.managers[label]
	s.mutex.RUnlock()

	if !ok {
		return cred.Identity{}
	}

	if identity, err := manager.Get(); err == nil {
		return identity
	}

	return cred.Identity{}
}

func (s *Service) GetReader(label credentials.Label) cred.IdentityReader {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.managers[label]
}
