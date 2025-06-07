package state

import "sync"

// StateManager interface defines the contract for state management
type StateManager interface {
	SetUserState(userID int64, state string)
	GetUserState(userID int64) string
	SetTempData(userID int64, key string, value interface{})
	GetTempData(userID int64, key string) (interface{}, bool)
	ClearTempData(userID int64)
	SetUserWeight(userID int64, weight float64)
	GetUserWeight(userID int64) float64
}

// User states constants
const (
	None                        = "none"
	WaitingForBloodSugar        = "waiting_for_blood_sugar"
	WaitingForInsulinRatio      = "waiting_for_insulin_ratio"
	WaitingForTimePeriod        = "waiting_for_time_period"
	WaitingForActiveInsulinTime = "waiting_for_active_insulin_time"
)

// InMemoryManager manages user states and temporary data in memory
type InMemoryManager struct {
	userStates  map[int64]string
	userWeights map[int64]float64
	tempData    map[int64]map[string]interface{}
	mu          sync.RWMutex
}

// NewInMemoryManager creates a new in-memory state manager
func NewInMemoryManager() *InMemoryManager {
	return &InMemoryManager{
		userStates:  make(map[int64]string),
		userWeights: make(map[int64]float64),
		tempData:    make(map[int64]map[string]interface{}),
	}
}

// SetUserState sets the state for a user
func (m *InMemoryManager) SetUserState(userID int64, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userStates[userID] = state
}

// GetUserState gets the state for a user
func (m *InMemoryManager) GetUserState(userID int64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, exists := m.userStates[userID]
	if !exists {
		return None
	}
	return state
}

// ClearUserState clears the state for a user
func (m *InMemoryManager) ClearUserState(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.userStates, userID)
}

// SetUserWeight sets the weight for a user
func (m *InMemoryManager) SetUserWeight(userID int64, weight float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userWeights[userID] = weight
}

// GetUserWeight gets the weight for a user - адаптирую под интерфейс
func (m *InMemoryManager) GetUserWeight(userID int64) float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	weight, exists := m.userWeights[userID]
	if !exists {
		return 0
	}
	return weight
}

// ClearUserWeight clears the weight for a user
func (m *InMemoryManager) ClearUserWeight(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.userWeights, userID)
}

// SetTempData sets temporary data for a user
func (m *InMemoryManager) SetTempData(userID int64, key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tempData[userID] == nil {
		m.tempData[userID] = make(map[string]interface{})
	}
	m.tempData[userID][key] = value
}

// GetTempData gets temporary data for a user
func (m *InMemoryManager) GetTempData(userID int64, key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	userData, exists := m.tempData[userID]
	if !exists {
		return nil, false
	}
	value, exists := userData[key]
	return value, exists
}

// ClearTempData clears all temporary data for a user
func (m *InMemoryManager) ClearTempData(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tempData, userID)
}
