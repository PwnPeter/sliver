package outlook

/*
	Sliver Implant Framework
	Copyright (C) 2024  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"sync"
	"time"
)

// ImplantSession représente une session d'implant
type ImplantSession struct {
	SessionID      string
	ImplantEmail   string
	LastSeen       time.Time
	PendingTasks   map[string]*Task
	CompletedTasks map[string]*TaskResult
	mutex          sync.RWMutex
}

// Task représente une tâche à exécuter
type Task struct {
	TaskID    string
	Command   string
	CreatedAt time.Time
	SentAt    *time.Time
	Status    string // "pending", "sent", "completed", "failed"
}

// TaskResult représente le résultat d'une tâche
type TaskResult struct {
	TaskID      string
	Result      string
	ReceivedAt  time.Time
	Error       string
	Success     bool
}

// SessionManager gère les sessions des implants
type SessionManager struct {
	sessions map[string]*ImplantSession
	mutex    sync.RWMutex
}

// NewSessionManager crée un nouveau gestionnaire de sessions
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ImplantSession),
	}
}

// GetOrCreateSession récupère ou crée une session
func (sm *SessionManager) GetOrCreateSession(sessionID, implantEmail string) *ImplantSession {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.LastSeen = time.Now()
		return session
	}

	session := &ImplantSession{
		SessionID:      sessionID,
		ImplantEmail:   implantEmail,
		LastSeen:       time.Now(),
		PendingTasks:   make(map[string]*Task),
		CompletedTasks: make(map[string]*TaskResult),
	}

	sm.sessions[sessionID] = session
	return session
}

// GetSession récupère une session par ID
func (sm *SessionManager) GetSession(sessionID string) *ImplantSession {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.sessions[sessionID]
}

// ListSessions retourne toutes les sessions
func (sm *SessionManager) ListSessions() []*ImplantSession {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	sessions := make([]*ImplantSession, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// RemoveSession supprime une session
func (sm *SessionManager) RemoveSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	delete(sm.sessions, sessionID)
}

// CleanupStaleSessions supprime les sessions inactives
func (sm *SessionManager) CleanupStaleSessions(timeout time.Duration) int {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	cleaned := 0
	now := time.Now()

	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastSeen) > timeout {
			delete(sm.sessions, sessionID)
			cleaned++
		}
	}

	return cleaned
}

// AddTask ajoute une tâche à une session
func (s *ImplantSession) AddTask(taskID, command string) *Task {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	task := &Task{
		TaskID:    taskID,
		Command:   command,
		CreatedAt: time.Now(),
		Status:    "pending",
	}

	s.PendingTasks[taskID] = task
	return task
}

// MarkTaskSent marque une tâche comme envoyée
func (s *ImplantSession) MarkTaskSent(taskID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if task, exists := s.PendingTasks[taskID]; exists {
		now := time.Now()
		task.SentAt = &now
		task.Status = "sent"
	}
}

// CompleteTask marque une tâche comme terminée
func (s *ImplantSession) CompleteTask(taskID, result string, success bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Retirer des tâches pendantes
	delete(s.PendingTasks, taskID)

	// Ajouter aux tâches terminées
	s.CompletedTasks[taskID] = &TaskResult{
		TaskID:     taskID,
		Result:     result,
		ReceivedAt: time.Now(),
		Success:    success,
	}

	s.LastSeen = time.Now()
}

// GetPendingTasks retourne les tâches en attente
func (s *ImplantSession) GetPendingTasks() []*Task {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tasks := make([]*Task, 0, len(s.PendingTasks))
	for _, task := range s.PendingTasks {
		tasks = append(tasks, task)
	}

	return tasks
}

// GetCompletedTasks retourne les tâches terminées
func (s *ImplantSession) GetCompletedTasks() []*TaskResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	results := make([]*TaskResult, 0, len(s.CompletedTasks))
	for _, result := range s.CompletedTasks {
		results = append(results, result)
	}

	return results
}

// GetTaskResult récupère le résultat d'une tâche
func (s *ImplantSession) GetTaskResult(taskID string) *TaskResult {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.CompletedTasks[taskID]
}

// IsActive vérifie si la session est active
func (s *ImplantSession) IsActive(timeout time.Duration) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return time.Since(s.LastSeen) < timeout
}
