package main

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

// inflightTTL bounds how long a request may stay counted as in flight. The host
// delivers request.complete asynchronously and best effort, so an entry whose
// completion never arrives, for example across a plugin reload, is dropped
// instead of inflating the concurrency figure forever.
const inflightTTL = 2 * time.Hour

// inflightRequest is one admitted request that has not completed yet.
type inflightRequest struct {
	KeyID   string
	Started time.Time
}

// beginRequest counts an admitted request as in flight for its key. Repeated
// calls with the same request ID, as happen on retries, count it once. The key
// is registered too, so its first request already shows on the dashboard.
func (s *store) beginRequest(requestID, id, hint string, at time.Time) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entryLocked(id, hint, true)
	s.beginRequestLocked(requestID, id, at)
}

func (s *store) beginRequestLocked(requestID, id string, at time.Time) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" || id == "" {
		return
	}
	if s.inflight == nil {
		s.inflight = map[string]inflightRequest{}
	}
	s.pruneInflightLocked(at)
	s.inflight[requestID] = inflightRequest{KeyID: id, Started: at}
}

// endRequest releases a request, and the account slot it held, once the host
// reports its terminal state, waking any request queued for a slot.
func (s *store) endRequest(requestID string) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, keySlot := s.inflight[requestID]
	_, accountSlot := s.accountInflight[requestID]
	delete(s.inflight, requestID)
	delete(s.accountInflight, requestID)
	if keySlot || accountSlot {
		s.notifyReleaseLocked()
	}
}

// pruneInflightLocked drops entries older than inflightTTL.
func (s *store) pruneInflightLocked(at time.Time) {
	for requestID, request := range s.inflight {
		if at.Sub(request.Started) > inflightTTL {
			delete(s.inflight, requestID)
		}
	}
}

// concurrencyLocked counts in-flight requests per key ID.
func (s *store) concurrencyLocked(at time.Time) map[string]int {
	s.pruneInflightLocked(at)
	counts := make(map[string]int, len(s.inflight))
	for _, request := range s.inflight {
		counts[request.KeyID]++
	}
	return counts
}

// handleRequestComplete releases the in-flight slot of a finished request.
func handleRequestComplete(s *store, raw []byte) error {
	var completion pluginapi.RequestCompletion
	if errUnmarshal := json.Unmarshal(raw, &completion); errUnmarshal != nil {
		return errUnmarshal
	}
	s.endRequest(completion.RequestID)
	return nil
}
