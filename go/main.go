package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	void* call;
	void* free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

var errInvalidPeriod = errors.New("period must be daily, monthly, total, or all")

var quotaStore = newStore(nil)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type lifecycleRequest struct {
	ConfigYAML    []byte `json:"config_yaml"`
	SchemaVersion uint32 `json:"schema_version"`
}

type registration struct {
	SchemaVersion uint32                 `json:"schema_version"`
	Metadata      pluginapi.Metadata     `json:"metadata"`
	Capabilities  registrationCapability `json:"capabilities"`
}

type registrationCapability struct {
	RequestInterceptor bool `json:"request_interceptor"`
	// Scheduler picks the account of keys bound to accounts and enforces
	// account concurrency. It receives candidates across every priority tier
	// so bound accounts are reachable whatever their priority.
	Scheduler                 bool `json:"scheduler"`
	SchedulerAcrossPriorities bool `json:"scheduler_across_priorities,omitempty"`
	RequestLifecyclePlugin    bool `json:"request_lifecycle_plugin"`
	UsagePlugin               bool `json:"usage_plugin"`
	ManagementAPI             bool `json:"management_api"`
	ModelRouter               bool `json:"model_router"`
	// Executor serves the requests model.route sends to the plugin itself,
	// which is how a key's fallback models are tried in turn.
	Executor              bool                         `json:"executor"`
	ExecutorModelScope    pluginapi.ExecutorModelScope `json:"executor_model_scope"`
	ExecutorInputFormats  []string                     `json:"executor_input_formats,omitempty"`
	ExecutorOutputFormats []string                     `json:"executor_output_formats,omitempty"`
}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	if host != nil && host.call != nil {
		setHostAPI(unsafe.Pointer(host))
	}
	plugin.abi_version = C.uint32_t(pluginabi.ABIVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, errHandle := handleMethod(C.GoString(method), requestBytes)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, len C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
	_ = len
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {
	quotaStore.flush()
}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		if errConfigure := configure(quotaStore, request); errConfigure != nil {
			return nil, errConfigure
		}
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodPluginShutdown, pluginabi.MethodPluginQuiesce:
		quotaStore.flush()
		return okEnvelope(struct{}{})
	case pluginabi.MethodModelRoute:
		return routeModel(quotaStore, request)
	case pluginabi.MethodRequestInterceptBefore:
		return interceptBeforeAuth(quotaStore, request)
	case pluginabi.MethodRequestInterceptAfter:
		return interceptAfterAuth(quotaStore, request)
	case pluginabi.MethodSchedulerPick:
		return schedulerPick(quotaStore, request)
	case pluginabi.MethodRequestComplete:
		if errComplete := handleRequestComplete(quotaStore, request); errComplete != nil {
			return nil, errComplete
		}
		return okEnvelope(struct{}{})
	case pluginabi.MethodUsageHandle:
		if errUsage := handleUsage(quotaStore, request); errUsage != nil {
			return nil, errUsage
		}
		return okEnvelope(struct{}{})
	case pluginabi.MethodExecutorIdentifier:
		return okEnvelope(map[string]string{"identifier": executorIdentifier})
	case pluginabi.MethodExecutorExecute:
		return executeWithFallback(quotaStore, request)
	case pluginabi.MethodExecutorExecuteStream:
		return executeStreamWithFallback(quotaStore, request)
	case pluginabi.MethodExecutorCountTokens:
		return countTokensUnsupported()
	case pluginabi.MethodExecutorHTTPRequest:
		return errorEnvelope("unsupported", "apikey-quota executor has no credentials to send HTTP requests with"), nil
	case pluginabi.MethodManagementRegister:
		return okEnvelope(managementRegistration())
	case pluginabi.MethodManagementHandle:
		return handleManagement(quotaStore, request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

// configure decodes the host lifecycle payload and applies it to the store.
func configure(s *store, raw []byte) error {
	var req lifecycleRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return errUnmarshal
		}
	}
	if req.SchemaVersion < 2 {
		return fmt.Errorf("apikey-quota requires host schema version 2 or newer")
	}
	cfg, errParse := parseConfig(req.ConfigYAML)
	if errParse != nil {
		return errParse
	}
	return s.configure(cfg)
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             "apikey-quota",
			Version:          "0.8.0",
			Author:           "minifun1218",
			GitHubRepository: "https://github.com/minifun1218/cliproxy-apikey-quota",
			Logo:             "https://raw.githubusercontent.com/minifun1218/cliproxy-apikey-quota/main/docs/logo.png",
			ConfigFields: []pluginapi.ConfigField{
				{
					Name:        "enforce",
					Type:        pluginapi.ConfigFieldTypeBoolean,
					Description: "Rejects requests once a quota is reached. Set to false to only observe usage.",
				},
				{
					Name:        "state_file",
					Type:        pluginapi.ConfigFieldTypeString,
					Description: "Path of the JSON file holding accumulated usage counters.",
				},
				{
					Name:        "persist_interval_seconds",
					Type:        pluginapi.ConfigFieldTypeInteger,
					Description: "Minimum delay between state file writes. Zero writes on every update.",
				},
				{
					Name:        "reject_status",
					Type:        pluginapi.ConfigFieldTypeInteger,
					Description: "HTTP status returned when a quota is exceeded.",
				},
				{
					Name:        "time_zone",
					Type:        pluginapi.ConfigFieldTypeString,
					Description: "IANA time zone deciding when daily and monthly periods roll over.",
				},
				{
					Name:        "count_failed_requests",
					Type:        pluginapi.ConfigFieldTypeBoolean,
					Description: "Charges failed upstream requests against the quota.",
				},
				{
					Name:        "track_unknown_keys",
					Type:        pluginapi.ConfigFieldTypeBoolean,
					Description: "Accounts for API keys that have no entry under keys.",
				},
				{
					Name:        "default_limits",
					Type:        pluginapi.ConfigFieldTypeObject,
					Description: "Daily, monthly, and total ceilings applied to keys without their own limits.",
				},
				{
					Name:        "default_rate_limits",
					Type:        pluginapi.ConfigFieldTypeObject,
					Description: "Requests per minute (rpm), tokens per minute (tpm), concurrent requests, and the seconds a request queues for a free slot (queue_seconds) applied to every key.",
				},
				{
					Name:        "default_schedule",
					Type:        pluginapi.ConfigFieldTypeObject,
					Description: "Time windows deciding when keys may be used, each with its own quota and rate limits.",
				},
				{
					Name:        "blocked_models",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Model name patterns every API key is forbidden to call.",
				},
				{
					Name:        "model_mappings",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Rules rewriting a requested model (from) to another model (to), optionally on a named provider.",
				},
				{
					Name:        "fallback_models",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Models tried in order when the requested model fails upstream, for keys without their own list.",
				},
				{
					Name:        "fallback_status_codes",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Upstream statuses that move a request on to the next fallback model. Defaults to 408, 429, 500, 502, 503, 504, and 529.",
				},
				{
					Name:        "keys",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Per API key label, note, disabled flag, blocked models, model mappings, fallback models, limit and rate limit overrides, schedule, and bound accounts.",
				},
				{
					Name:        "accounts",
					Type:        pluginapi.ConfigFieldTypeArray,
					Description: "Rules matching host accounts by ID pattern (match), capping their concurrent requests (concurrency) and reserving them for the keys bound to them (exclusive).",
				},
				{
					Name:        "pricing",
					Type:        pluginapi.ConfigFieldTypeObject,
					Description: "USD prices per one million tokens used to compute cost quotas.",
				},
			},
		},
		Capabilities: registrationCapability{
			RequestInterceptor:        true,
			Scheduler:                 true,
			SchedulerAcrossPriorities: true,
			RequestLifecyclePlugin:    true,
			UsagePlugin:               true,
			ManagementAPI:             true,
			ModelRouter:               true,
			Executor:                  true,
			ExecutorModelScope:        pluginapi.ExecutorModelScopeStatic,
			ExecutorInputFormats:      executorFormats,
			ExecutorOutputFormats:     executorFormats,
		},
	}
}

func okEnvelope(v any) ([]byte, error) {
	raw, errMarshal := json.Marshal(v)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, errMarshal := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	if errMarshal != nil {
		return []byte(`{"ok":false,"error":{"code":"plugin_error","message":"encode error"}}`)
	}
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}

// hostLogf reports plugin-side problems on the host process stderr.
func hostLogf(format string, args ...any) {
	if _, errWrite := fmt.Fprintf(os.Stderr, format+"\n", args...); errWrite != nil {
		return
	}
}
