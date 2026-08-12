package whatsapp

import (
	"errors"
	"testing"

	"go.mau.fi/whatsmeow"
	waLog "go.mau.fi/whatsmeow/util/log"
)

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
	unlinkedUser := ""
	cm.SetOnUnlink(func(user string) {
		unlinkedUser = user
	})
	cm.handleRuntimeDisconnect("628123456789", "logged_out", true, true)

	if _, ok := cm.clients["628123456789"]; ok {
		t.Fatal("runtime disconnect did not remove client")
	}
	if unlinkedUser != "628123456789" {
		t.Fatalf("expected unlinkedUser to be 628123456789, got %q", unlinkedUser)
	}
}

func TestEnsureClientDoesNotConnectExistingDisconnectedClient(t *testing.T) {
	cm := &ClientManager{
		clients:       make(map[string]*whatsmeow.Client),
		groupsCache:   make(map[string]groupCacheEntry),
		contactsCache: make(map[string]contactCacheEntry),
		log:           waLog.Noop,
	}
	cm.clients["628123456789"] = &whatsmeow.Client{}

	if err := cm.EnsureClient("628123456789"); err != nil {
		t.Fatalf("EnsureClient() = %v, want nil while whatsmeow owns reconnect", err)
	}
}
