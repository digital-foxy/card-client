package keyringcred

import (
	"errors"
	"sync"
	"testing"

	"github.com/digital-foxy/card-client/credentials"
	"github.com/digital-foxy/toolkit/cred"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	LabelServiceA credentials.Label = "service-a"
	LabelServiceB credentials.Label = "service-b"
	LabelServiceC credentials.Label = "service-c"
)

func init() {
	credentials.Labels = []credentials.Label{LabelServiceA, LabelServiceB}
}

type MockManager struct {
	mock.Mock
}

func (m *MockManager) CredLabel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockManager) Get() (cred.Identity, error) {
	args := m.Called()
	return args.Get(0).(cred.Identity), args.Error(1)
}
func (m *MockManager) GetUser() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}
func (m *MockManager) GetSecret() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}
func (m *MockManager) SetAll(identity cred.Identity) error {
	return m.Called(identity).Error(0)
}
func (m *MockManager) Set(payload cred.IdentityPayload) error {
	return m.Called(payload).Error(0)
}
func (m *MockManager) SetUser(user string) error {
	return m.Called(user).Error(0)
}
func (m *MockManager) SetSecret(secret string) error {
	return m.Called(secret).Error(0)
}
func (m *MockManager) Delete() error {
	return m.Called().Error(0)
}

func setupTestServiceWithMocks(t *testing.T) (*Service, map[credentials.Label]*MockManager) {
	t.Helper()
	s := NewService()
	mocks := make(map[credentials.Label]*MockManager)

	for _, label := range s.labels {
		mockManager := new(MockManager)
		s.managers[label] = mockManager
		mocks[label] = mockManager
	}
	return s, mocks
}

func TestNewService(t *testing.T) {
	s := NewService()
	require.NotNil(t, s)
	assert.Len(t, s.managers, len(credentials.Labels))
	for _, label := range credentials.Labels {
		assert.NotNil(t, s.managers[label])
	}
}

func TestService_Labels(t *testing.T) {
	s, _ := setupTestServiceWithMocks(t)
	assert.Equal(t, credentials.Labels, s.Labels())
}

func TestService_RegisterLabel(t *testing.T) {
	s := NewService()
	initialCount := len(s.Labels())

	s.RegisterLabel(LabelServiceC)
	assert.Len(t, s.Labels(), initialCount+1)
	assert.NotNil(t, s.managers[LabelServiceC])

	s.RegisterLabel(LabelServiceA)
	assert.Len(t, s.Labels(), initialCount+1)
}

func TestService_IdentityMethods(t *testing.T) {
	s, mocks := setupTestServiceWithMocks(t)
	mockManagerGH := mocks[LabelServiceA]
	mockManagerGL := mocks[LabelServiceB]
	user := "user"
	secret := "secret"
	payload := cred.IdentityPayload{User: &user, Secret: &secret}

	t.Run("GetIdentity", func(t *testing.T) {
		identity := cred.Identity{User: "user-gh", Secret: "secret-gh"}
		mockManagerGH.On("Get").Return(identity, nil).Once()
		result := s.GetIdentity(LabelServiceA)
		assert.Equal(t, identity, result)
		mockManagerGH.AssertExpectations(t)
	})

	t.Run("GetIdentity returns empty on error", func(t *testing.T) {
		mockManagerGL.On("Get").Return(cred.Identity{}, errors.New("fail")).Once()
		result := s.GetIdentity(LabelServiceB)
		assert.Equal(t, cred.Identity{}, result)
		mockManagerGL.AssertExpectations(t)
	})

	t.Run("GetIdentity returns empty for unknown label", func(t *testing.T) {
		result := s.GetIdentity(LabelServiceC)
		assert.Equal(t, cred.Identity{}, result)
	})

	t.Run("SetIdentity", func(t *testing.T) {
		mockManagerGH.On("Set", payload).Return(nil).Once()
		s.SetIdentity(LabelServiceA, payload)
		mockManagerGH.AssertExpectations(t)
	})

	t.Run("SetIdentity ignores unknown label", func(t *testing.T) {
		assert.NotPanics(t, func() {
			s.SetIdentity(LabelServiceC, payload)
		})
	})
}

func TestService_BulkIdentityMethods(t *testing.T) {
	s, mocks := setupTestServiceWithMocks(t)
	mockManagerGH := mocks[LabelServiceA]
	mockManagerGL := mocks[LabelServiceB]
	payloadA := cred.IdentityPayload{User: &[]string{"user-a"}[0]}
	payloadB := cred.IdentityPayload{Secret: &[]string{"secret-b"}[0]}
	payloadC := cred.IdentityPayload{User: &[]string{"user-c"}[0]}

	t.Run("GetIdentities", func(t *testing.T) {
		identityGH := cred.Identity{User: "user-gh"}
		mockManagerGH.On("Get").Return(identityGH, nil).Once()
		mockManagerGL.On("Get").Return(cred.Identity{}, errors.New("fail")).Once()
		identities := s.GetIdentities()
		assert.Len(t, identities, 1)
		assert.Equal(t, identityGH, identities[LabelServiceA])
		mockManagerGH.AssertExpectations(t)
		mockManagerGL.AssertExpectations(t)
	})

	t.Run("SetIdentities", func(t *testing.T) {
		payloadMap := map[credentials.Label]cred.IdentityPayload{
			LabelServiceA: payloadA,
			LabelServiceB: payloadB,
			LabelServiceC: payloadC,
		}
		mockManagerGH.On("Set", payloadA).Return(nil).Once()
		mockManagerGL.On("Set", payloadB).Return(nil).Once()
		s.SetIdentities(payloadMap)
		mockManagerGH.AssertExpectations(t)
		mockManagerGL.AssertExpectations(t)
	})
}

func TestService_GetUsers(t *testing.T) {
	s, mocks := setupTestServiceWithMocks(t)
	mockManagerA := mocks[LabelServiceA]
	mockManagerB := mocks[LabelServiceB]

	expectedUserA := "test-user-a"

	mockManagerA.On("GetUser").Return(expectedUserA, nil).Once()
	mockManagerB.On("GetUser").Return("", errors.New("failed to get user")).Once()

	users := s.GetUsers()

	assert.Len(t, users, 1, "Should only return users for which the call was successful")
	assert.Equal(t, expectedUserA, users[LabelServiceA], "The user for Service A should match the expected user")
	_, ok := users[LabelServiceB]
	assert.False(t, ok, "Service B should not be in the users map because it returned an error")

	mockManagerA.AssertExpectations(t)
	mockManagerB.AssertExpectations(t)
}

func TestService_GetReader(t *testing.T) {
	s, mocks := setupTestServiceWithMocks(t)
	mockManagerGH := mocks[LabelServiceA]

	reader := s.GetReader(LabelServiceA)
	assert.Equal(t, mockManagerGH, reader)

	reader = s.GetReader(LabelServiceC)
	assert.Nil(t, reader)
}

func TestService_Concurrency(t *testing.T) {
	s, mocks := setupTestServiceWithMocks(t)
	mockManagerGH := mocks[LabelServiceA]
	mockManagerGL := mocks[LabelServiceB]

	mockManagerGH.On("Get").Return(cred.Identity{}, nil)
	mockManagerGL.On("Set", mock.Anything).Return(nil)
	mockManagerGL.On("Get").Return(cred.Identity{}, nil)

	var wg sync.WaitGroup
	numGoroutines := 100

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			s.Labels()
			s.RegisterLabel("new-label")
			s.GetIdentity("service-a")
			s.SetIdentity("service-b", cred.IdentityPayload{})
			s.GetIdentities()
		}()
	}
	wg.Wait()
}
