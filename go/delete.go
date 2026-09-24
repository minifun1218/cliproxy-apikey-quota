package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const routeDelete = routePrefix + "/delete"

// deleteRequest drops the accounting records of one or several keys. It does
// not change the host's key list: the dashboard removes a key from the host
// first, through the host's own api-keys management route, and then calls
// this route.
type deleteRequest struct {
	ID  string   `json:"id"`
	IDs []string `json:"ids"`
}

type deleteResult struct {
	// Deleted lists the ids whose usage and settings were removed.
	Deleted []string `json:"deleted"`
	// Configured lists the ids that also have an entry under keys in the plugin
	// configuration. They stay listed with zeroed usage until that entry is
	// removed by hand.
	Configured []string `json:"configured"`
	// Missing lists the ids that were neither tracked nor configured.
	Missing []string `json:"missing"`
}

// deleteKeys removes the usage, limits, note, deny list, and mappings of each
// key. Live concurrency is left alone: in-flight requests finish and are
// metered as a fresh record.
func (s *store) deleteKeys(ids []string) deleteResult {
	result := deleteResult{Deleted: []string{}, Configured: []string{}, Missing: []string{}}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		_, tracked := s.state.Keys[id]
		_, configured := s.cfg.byID[id]
		if !tracked && !configured {
			result.Missing = append(result.Missing, id)
			continue
		}
		delete(s.state.Keys, id)
		delete(s.rates, id)
		result.Deleted = append(result.Deleted, id)
		if configured {
			result.Configured = append(result.Configured, id)
		}
	}
	if len(result.Deleted) > 0 {
		s.dirty = true
		s.persistLocked(true)
	}
	return result
}

func handleDelete(s *store, req pluginapi.ManagementRequest) ([]byte, error) {
	var body deleteRequest
	if errUnmarshal := json.Unmarshal(req.Body, &body); errUnmarshal != nil {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
	}
	ids := targetIDs(strings.TrimSpace(body.ID), body.IDs)
	if len(ids) == 0 {
		return jsonResponse(http.StatusBadRequest, map[string]string{"error": "id or ids is required"})
	}
	return jsonResponse(http.StatusOK, s.deleteKeys(ids))
}
