package state

import "sync"

// User states constants
const (
	None                        = "none"
	WaitingForBloodSugar        = "waiting_for_blood_sugar"
	WaitingForInsulinRatio      = "waiting_for_insulin_ratio"
	WaitingForTimePeriod        = "waiting_for_time_period"
	WaitingForActiveInsulinTime = "waiting_for_active_insulin_time"
)

// Manager manages user states and temporary data
type Manager struct {
	userStates  map[int64]string
	userWeights map[int64]float64
	tempData    map[int64]map[string]interface{}
	mu          sync.RWMutex
}

// NewManager creates a new state manager
func NewManager() *Manager {
	return &Manager{
		userStates:  make(map[int64]string),
		userWeights: make(map[int64]float64),
		tempData:    make(map[int64]map[string]interface{}),
	}
}

// SetUserState sets the state for a user
func (m *Manager) SetUserState(userID int64, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userStates[userID] = state
}

// GetUserState gets the state for a user
func (m *Manager) GetUserState(userID int64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	state, exists := m.userStates[userID]
	if !exists {
		return None
	}
	return state
}

// ClearUserState clears the state for a user
func (m *Manager) ClearUserState(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.userStates, userID)
}

// SetUserWeight sets the weight for a user
func (m *Manager) SetUserWeight(userID int64, weight float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userWeights[userID] = weight
}

// GetUserWeight gets the weight for a user
func (m *Manager) GetUserWeight(userID int64) (float64, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	weight, exists := m.userWeights[userID]
	return weight, exists
}

// ClearUserWeight clears the weight for a user
func (m *Manager) ClearUserWeight(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.userWeights, userID)
}

// SetTempData sets temporary data for a user
func (m *Manager) SetTempData(userID int64, key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.tempData[userID] == nil {
		m.tempData[userID] = make(map[string]interface{})
	}
	m.tempData[userID][key] = value
}

// GetTempData gets temporary data for a user
func (m *Manager) GetTempData(userID int64, key string) (interface{}, bool) {
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
func (m *Manager) ClearTempData(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tempData, userID)
}
