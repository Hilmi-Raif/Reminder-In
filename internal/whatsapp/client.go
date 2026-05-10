package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type ClientManager struct {
	container *sqlstore.Container
	clients   map[string]*whatsmeow.Client
	mu        sync.RWMutex
	log       waLog.Logger

	sendTimeout   time.Duration
	queryTimeout  time.Duration
	cacheTTL      time.Duration
	groupsCache   map[string]groupCacheEntry
	contactsCache map[string]contactCacheEntry
}

type groupCacheEntry struct {
	data      []GroupInfo
	expiresAt time.Time
}

type contactCacheEntry struct {
	data      []ContactInfo
	expiresAt time.Time
}

func NewClientManager() (*ClientManager, error) {
	dbLog := waLog.Stdout("Database", "WARN", true)
	clientLog := waLog.Stdout("Client", "WARN", true)

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/reminderin.db"
	}
	dbDir := filepath.Dir(dbPath)
	waDbPath := filepath.Join(dbDir, "wa_sessions.db")
	waDbPath = strings.ReplaceAll(waDbPath, "\\", "/")

	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	container, err := sqlstore.New(context.Background(), "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", waDbPath), dbLog)
	if err != nil {
		return nil, err
	}

	sendTimeout := durationFromEnvSeconds("WA_SEND_TIMEOUT_SECONDS", 20)
	queryTimeout := durationFromEnvSeconds("WA_QUERY_TIMEOUT_SECONDS", 20)
	cacheTTL := durationFromEnvSeconds("WA_DIRECTORY_CACHE_TTL_SECONDS", 60)

	return &ClientManager{
		container:     container,
		clients:       make(map[string]*whatsmeow.Client),
		log:           clientLog,
		sendTimeout:   sendTimeout,
		queryTimeout:  queryTimeout,
		cacheTTL:      cacheTTL,
		groupsCache:   make(map[string]groupCacheEntry),
		contactsCache: make(map[string]contactCacheEntry),
	}, nil
}

func (cm *ClientManager) GetClient(jid string) (*whatsmeow.Client, error) {
	cm.mu.RLock()
	client, ok := cm.clients[jid]
	cm.mu.RUnlock()
	if ok {
		return client, nil
	}

	return nil, fmt.Errorf("client not initialized for %s", jid)
}

func (cm *ClientManager) GetNewAuthClient() *whatsmeow.Client {
	deviceStore := cm.container.NewDevice()
	client := whatsmeow.NewClient(deviceStore, cm.log)
	cm.setupEventHandler(client)
	return client
}

func (cm *ClientManager) setupEventHandler(client *whatsmeow.Client) {
	client.AddEventHandler(func(evt interface{}) {
		switch e := evt.(type) {
		case *events.LoggedOut:
			if e.OnConnect && e.Reason.IsLoggedOut() {
				if client.Store.ID != nil {
					user := client.Store.ID.User
					cm.mu.Lock()
					delete(cm.clients, user)
					delete(cm.groupsCache, user)
					delete(cm.contactsCache, user)
					cm.mu.Unlock()
					cm.log.Warnf("Client %s logged out (OnConnect: %v, Reason: %v)", user, e.OnConnect, e.Reason)
				}
			} else if !e.OnConnect {
				if client.Store.ID != nil {
					user := client.Store.ID.User
					cm.mu.Lock()
					delete(cm.clients, user)
					delete(cm.groupsCache, user)
					delete(cm.contactsCache, user)
					cm.mu.Unlock()
					cm.log.Warnf("Client %s logged out via stream error", user)
				}
			}
		}
	})
}

func (cm *ClientManager) LoadAllClients() error {
	devices, err := cm.container.GetAllDevices(context.Background())
	if err != nil {
		return err
	}

	for _, device := range devices {
		client := whatsmeow.NewClient(device, cm.log)
		cm.setupEventHandler(client)
		if device.ID != nil {
			cm.mu.Lock()
			cm.clients[device.ID.User] = client
			delete(cm.groupsCache, device.ID.User)
			delete(cm.contactsCache, device.ID.User)
			cm.mu.Unlock()
		}
		if err := client.Connect(); err != nil {
			continue
		}
	}

	return nil
}

func (cm *ClientManager) LoadClient(user string) error {
	user = strings.TrimSpace(user)
	if user == "" {
		return nil
	}

	devices, err := cm.container.GetAllDevices(context.Background())
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.ID == nil || device.ID.User != user {
			continue
		}

		client := whatsmeow.NewClient(device, cm.log)
		cm.setupEventHandler(client)
		cm.mu.Lock()
		cm.clients[user] = client
		delete(cm.groupsCache, user)
		delete(cm.contactsCache, user)
		cm.mu.Unlock()

		if err := client.Connect(); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("client session not found for %s", user)
}

func (cm *ClientManager) SendMessage(jid string, target string, message string) error {
	cm.mu.RLock()
	client, ok := cm.clients[jid]
	cm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("client not found for JID: %s", jid)
	}

	if !client.IsConnected() {
		_ = client.Connect()
		if !client.IsConnected() {
			return fmt.Errorf("client %s is not connected", jid)
		}
	}

	if !strings.Contains(target, "@") {
		if strings.Contains(target, "-") {
			target = target + "@g.us"
		} else {
			target = target + "@s.whatsapp.net"
		}
	}

	targetJID, err := types.ParseJID(target)
	if err != nil {
		return err
	}

	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	ctx := context.Background()
	if cm.sendTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cm.sendTimeout)
		defer cancel()
	}

	_, err = client.SendMessage(ctx, targetJID, msg)
	return err
}

func (cm *ClientManager) AddClient(client *whatsmeow.Client) {
	if client.Store.ID != nil {
		cm.mu.Lock()
		cm.clients[client.Store.ID.User] = client
		delete(cm.groupsCache, client.Store.ID.User)
		delete(cm.contactsCache, client.Store.ID.User)
		cm.mu.Unlock()
	}
}

func (cm *ClientManager) SendPresence(jid string) error {
	cm.mu.RLock()
	client, ok := cm.clients[jid]
	cm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("client not found for JID: %s", jid)
	}

	if !client.IsConnected() {
		_ = client.Connect()
		if !client.IsConnected() {
			return fmt.Errorf("client %s is not connected", jid)
		}
	}

	ctx := context.Background()
	if cm.sendTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cm.sendTimeout)
		defer cancel()
	}

	return client.SendPresence(ctx, types.PresenceAvailable)
}

func (cm *ClientManager) Logout(jid string) error {
	cm.mu.RLock()
	client, ok := cm.clients[jid]
	cm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("client not found for JID: %s", jid)
	}

	logoutErr := client.Logout(context.Background())
	cm.mu.Lock()
	delete(cm.clients, jid)
	delete(cm.groupsCache, jid)
	delete(cm.contactsCache, jid)
	cm.mu.Unlock()
	if logoutErr != nil {
		return logoutErr
	}
	return nil
}

type GroupInfo struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

func (cm *ClientManager) GetJoinedGroups(jid string) ([]GroupInfo, error) {
	cm.mu.RLock()
	client, ok := cm.clients[jid]
	cacheEntry, hasCache := cm.groupsCache[jid]
	cacheTTL := cm.cacheTTL
	queryTimeout := cm.queryTimeout
	cm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("client not found for JID: %s", jid)
	}

	if !client.IsConnected() {
		_ = client.Connect()
		if !client.IsConnected() {
			return nil, fmt.Errorf("client %s is not connected", jid)
		}
	}

	now := time.Now()
	if cacheTTL > 0 && hasCache && cacheEntry.expiresAt.After(now) {
		return cloneGroupInfos(cacheEntry.data), nil
	}

	ctx := context.Background()
	if queryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, queryTimeout)
		defer cancel()
	}

	groups, err := client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]GroupInfo, 0, len(groups))
	for _, g := range groups {
		result = append(result, GroupInfo{
			JID:  g.JID.String(),
			Name: g.Name,
		})
	}

	if cacheTTL > 0 {
		cm.mu.Lock()
		cm.groupsCache[jid] = groupCacheEntry{
			data:      result,
			expiresAt: time.Now().Add(cacheTTL),
		}
		cm.mu.Unlock()
	}

	return result, nil
}

type ContactInfo struct {
	JID  string `json:"jid"`
	Name string `json:"name"`
}

func (cm *ClientManager) GetContacts(jid string) ([]ContactInfo, error) {
	cm.mu.RLock()
	client, ok := cm.clients[jid]
	cacheEntry, hasCache := cm.contactsCache[jid]
	cacheTTL := cm.cacheTTL
	queryTimeout := cm.queryTimeout
	cm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("client not found for JID: %s", jid)
	}

	if !client.IsConnected() {
		cm.log.Warnf("Client %s is not connected, attempting to fetch contacts from local store", jid)
	}

	now := time.Now()
	if cacheTTL > 0 && hasCache && cacheEntry.expiresAt.After(now) {
		return cloneContactInfos(cacheEntry.data), nil
	}

	ctx := context.Background()
	if queryTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, queryTimeout)
		defer cancel()
	}

	contacts, err := client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]ContactInfo, 0, len(contacts))
	for contactJID, info := range contacts {
		if contactJID.Server != "s.whatsapp.net" {
			continue
		}
		name := info.PushName
		if name == "" {
			name = info.FullName
		}
		if name == "" {
			name = contactJID.User
		}
		result = append(result, ContactInfo{
			JID:  contactJID.User,
			Name: name,
		})
	}

	slices.SortFunc(result, func(a, b ContactInfo) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})

	if cacheTTL > 0 {
		cm.mu.Lock()
		cm.contactsCache[jid] = contactCacheEntry{
			data:      result,
			expiresAt: time.Now().Add(cacheTTL),
		}
		cm.mu.Unlock()
	}

	return result, nil
}

func (cm *ClientManager) GetLinkQR(client *whatsmeow.Client) (<-chan whatsmeow.QRChannelItem, error) {
	if client.Store.ID == nil {
		qrChan, err := client.GetQRChannel(context.Background())
		if err != nil {
			return nil, err
		}
		err = client.Connect()
		if err != nil {
			return nil, err
		}
		return qrChan, nil
	}
	return nil, fmt.Errorf("client already has an ID")
}

func (cm *ClientManager) GetLinkCode(client *whatsmeow.Client, phone string) (string, error) {
	if client.Store.ID == nil {
		err := client.Connect()
		if err != nil {
			return "", err
		}

		code, err := client.PairPhone(context.Background(), phone, true, whatsmeow.PairClientChrome, "Chrome (Windows)")
		if err != nil {
			return "", err
		}
		return code, nil
	}
	return "", fmt.Errorf("client already has an ID")
}

func durationFromEnvSeconds(key string, fallbackSeconds int) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return time.Duration(fallbackSeconds) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func cloneGroupInfos(in []GroupInfo) []GroupInfo {
	if len(in) == 0 {
		return []GroupInfo{}
	}
	out := make([]GroupInfo, len(in))
	copy(out, in)
	return out
}

func cloneContactInfos(in []ContactInfo) []ContactInfo {
	if len(in) == 0 {
		return []ContactInfo{}
	}
	out := make([]ContactInfo, len(in))
	copy(out, in)
	return out
}
