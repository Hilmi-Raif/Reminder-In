package whatsapp

import (
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func TestIsFatalConnectErrorTreatsNetworkErrorsAsNonFatal(t *testing.T) {
	nonFatalErrors := []error{
		errors.New("lookup web.whatsapp.com on 127.0.0.11:53: read udp 172.19.0.2:44075->127.0.0.11:53: i/o timeout"),
		errors.New("dial tcp: lookup web.whatsapp.com: no such host"),
		errors.New("dial tcp 1.2.3.4:443: connect: connection refused"),
		errors.New("websocket: close 1006 (abnormal closure): unexpected EOF"),
		errors.New("Client outdated (405) connect failure"),
		errors.New("client is out of date"),
	}

	for _, err := range nonFatalErrors {
		if isFatalConnectError(err) {
			t.Fatalf("isFatalConnectError(%q) = true, want false", err.Error())
		}
	}
}

func TestIsFatalConnectErrorTreatsAccountErrorsAsFatal(t *testing.T) {
	fatalErrors := []error{
		errors.New("server returned 401"),
		errors.New("server returned 403"),
		errors.New("logged out from WhatsApp"),
		errors.New("stream error: device_removed"),
		errors.New("unofficial app detected"),
	}

	for _, err := range fatalErrors {
		if !isFatalConnectError(err) {
			t.Fatalf("isFatalConnectError(%q) = false, want true", err.Error())
		}
	}
}

func TestIsClientOutdatedConnectError(t *testing.T) {
	outdatedErrors := []error{
		errors.New("Client outdated (405) connect failure"),
		errors.New("server returned 405"),
		errors.New("client is out of date"),
	}

	for _, err := range outdatedErrors {
		if !isClientOutdatedConnectError(err) {
			t.Fatalf("isClientOutdatedConnectError(%q) = false, want true", err.Error())
		}
		if isFatalConnectError(err) {
			t.Fatalf("isFatalConnectError(%q) = true, want false", err.Error())
		}
	}
}

func TestRuntimeDisconnectRemovesOnlyRuntimeClient(t *testing.T) {
	cm := &ClientManager{
		clients:       make(map[string]*whatsmeow.Client),
		groupsCache:   make(map[string]groupCacheEntry),
		contactsCache: make(map[string]contactCacheEntry),
		log:           waLog.Noop,
	}

	cm.clients["628123456789"] = &whatsmeow.Client{}
	cm.handleRuntimeDisconnect("628123456789", "logged_out", true)

	if _, ok := cm.clients["628123456789"]; ok {
		t.Fatal("runtime disconnect did not remove client")
	}
}
