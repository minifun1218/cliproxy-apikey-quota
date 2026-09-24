package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} hc_buffer;

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	void* call;
	void* free_buffer;
} hc_host_api;

typedef int (*hc_call_fn)(void*, char*, uint8_t*, size_t, hc_buffer*);
typedef void (*hc_free_fn)(void*, size_t);

static int hc_call(void* api, char* method, uint8_t* request, size_t request_len, hc_buffer* response) {
	hc_host_api* host = (hc_host_api*)api;
	if (host == NULL || host->call == NULL) {
		return 1;
	}
	return ((hc_call_fn)host->call)(host->host_ctx, method, request, request_len, response);
}

static void hc_free(void* api, void* ptr, size_t len) {
	hc_host_api* host = (hc_host_api*)api;
	if (host == NULL || host->free_buffer == NULL || ptr == NULL) {
		return;
	}
	((hc_free_fn)host->free_buffer)(ptr, len);
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"unsafe"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
)

// hostAPI is the host function table handed to cliproxy_plugin_init. The host
// keeps it alive for as long as the library stays loaded.
var hostAPI atomic.Pointer[byte]

// setHostAPI records the host function table for later callbacks.
func setHostAPI(api unsafe.Pointer) {
	hostAPI.Store((*byte)(api))
}

// hostAvailable reports whether the host handed over a callable function table.
func hostAvailable() bool {
	return hostAPI.Load() != nil
}

// hostError is a failed host callback, carrying the HTTP status the host
// attached to it, or zero when it named none.
type hostError struct {
	Code    string
	Message string
	Status  int
}

func (e *hostError) Error() string {
	if e.Status > 0 {
		return fmt.Sprintf("%s (status %d)", e.Message, e.Status)
	}
	return e.Message
}

var errHostUnavailable = errors.New("host callbacks are unavailable")

// callHost sends one RPC to the host and decodes its result into out.
func callHost(method string, request any, out any) error {
	api := hostAPI.Load()
	if api == nil {
		return errHostUnavailable
	}
	raw, errMarshal := json.Marshal(request)
	if errMarshal != nil {
		return errMarshal
	}

	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	var cRequest *C.uint8_t
	if len(raw) > 0 {
		cRequest = (*C.uint8_t)(C.CBytes(raw))
		defer C.free(unsafe.Pointer(cRequest))
	}
	var response C.hc_buffer
	rc := C.hc_call(unsafe.Pointer(api), cMethod, cRequest, C.size_t(len(raw)), &response)
	var body []byte
	if response.ptr != nil && response.len > 0 {
		body = C.GoBytes(response.ptr, C.int(response.len))
		C.hc_free(unsafe.Pointer(api), response.ptr, response.len)
	}
	if rc != 0 && len(body) == 0 {
		return fmt.Errorf("host callback %s failed with code %d", method, int(rc))
	}

	var envelope pluginabi.Envelope
	if errUnmarshal := json.Unmarshal(body, &envelope); errUnmarshal != nil {
		return fmt.Errorf("decoding host callback %s: %w", method, errUnmarshal)
	}
	if !envelope.OK {
		if envelope.Error == nil {
			return &hostError{Code: "host_call_failed", Message: "host callback " + method + " failed"}
		}
		return &hostError{Code: envelope.Error.Code, Message: envelope.Error.Message, Status: envelope.Error.HTTPStatus}
	}
	if out == nil || len(envelope.Result) == 0 {
		return nil
	}
	return json.Unmarshal(envelope.Result, out)
}
