package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const routeRotate = routePrefix + "/rotate"

var (
	errRotateSameKey = errors.New("new_key must differ from the key being replaced")
	errRotateTaken   = errors.New("new_key is already tracked by another entry")
	errRotateUnknown = errors.New("no tracked or configured key has this id")
)

// rotateRequest moves the accounting record of one key to its replacement. The
// dashboard replaces the key in the host first, through the host's own
// api-keys management route, and then calls this route. NewKey is only hashed
// and masked here; the raw value is never stored.
type rotateRequest struct {
	ID     string `json:"id"`
	NewKey string `json:"new_key"`
}

type rotateResult struct {
	ID   string `json:"id"`
	Hint string `json:"hint"`
	// Configured reports that the old key also had an entry under keys in the
	// plugin configuration. Its settings now live on the new key as runtime
	// overrides, and that entry should be updated or removed by hand.
	Configured bool `json:"configured"`
}

// rotateKey re-keys the state of one API key so its usage, limits, note, deny
// list, and mappings follow it to the new secret. Settings that only existed in
// the configuration are copied in as runtime overrides, because the configured
// entry is bound to the old key and no longer applies once it is replaced.
func (s *store) rotateKey(req rotateRequest) (rotateResult, error) {
	newKey := strings.TrimSpace(req.NewKey)
	newID := keyID(newKey)
	if newID == req.ID {
		return rotateResult{}, errRotateSameKey
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.state.Keys[req.ID]
	configured, hasConfig := s.cfg.byID[req.ID]
	if entry == nil && !hasConfig {
		return rotateResult{}, errRotateUnknown
	}
	if _, taken := s.state.Keys[newID]; taken {
		return rotateResult{}, errRotateTaken
	}
	if _, taken := s.cfg.byID[newID]; taken {
		return rotateResult{}, errRotateTaken
	}
	if entry == nil {
		entry = &keyState{}
	}

	if hasConfig {
		effective, disabled := s.effectiveLimitsLocked(req.ID, entry)
		entry.Overrides = &effective
		if entry.Disabled == nil {
			entry.Disabled = &disabled
		}
		if entry.BlockedModels == nil {
			patterns := append([]string{}, configured.BlockedModels...)
			entry.BlockedModels = &patterns
		}
		if entry.ModelMappings == nil {
			mappings := append([]modelMapping{}, configured.ModelMappings...)
			entry.ModelMappings = &mappings
		}
		if entry.FallbackModels == nil && configured.FallbackModels != nil {
			models := append([]string{}, configured.FallbackModels...)
			entry.FallbackModels = &models
		}
		if entry.Note == nil {
			note := configured.Note
			entry.Note = &note
		}
		rate := s.baseRateLocked(req.ID, entry)
		entry.RateOverrides = &rate
		if entry.Schedule == nil && configured.Schedule != nil {
			entry.Schedule = cloneSchedule(configured.Schedule)
		}
		if entry.Accounts == nil && configured.Accounts != nil {
			accounts := append([]string{}, configured.Accounts...)
			entry.Accounts = &accounts
		}
		if entry.StrictAccounts == nil && configured.StrictAccounts {
			strict := true
			entry.StrictAccounts = &strict
		}
		// A label that is only the masked old key would now be misleading.
		if configured.Label != maskKey(configured.Key) {
			entry.Label = configured.Label
		}
	}
	entry.Hint = maskKey(newKey)

	delete(s.state.Keys, req.ID)
	s.state.Keys[newID] = entry
	if track, ok := s.rates[req.ID]; ok {
		delete(s.rates, req.ID)
		s.rates[newID] = track
	}
	s.dirty = true
	s.persistLocked(true)
	return rotateResult{ID: newID, Hint: entry.Hint, Configured: hasConfig}, nil
}

func handleRotate(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body rotateRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	body.ID = strings.TrimSpace(body.ID)
	if body.ID == "" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "id is required"})
	}
	if strings.TrimSpace(body.NewKey) == "" {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "new_key is required"})
	}
	result, errRotate := s.rotateKey(body)
	if errRotate != nil {
		status := http.StatusBadRequest
		if errors.Is(errRotate, errRotateUnknown) {
			status = http.StatusNotFound
		} else if errors.Is(errRotate, errRotateTaken) {
			status = http.StatusConflict
		}
		return jsonResponse(status, map[string]string{"error": errRotate.Error()})
	}
	return jsonResponse(http.StatusOK, result)
}
