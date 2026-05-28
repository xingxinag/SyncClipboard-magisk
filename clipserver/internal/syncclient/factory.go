package syncclient

import (
	"fmt"

	"github.com/yourusername/syncclipboard-android/clipserver/internal/config"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/serverclient"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/sync"
	"github.com/yourusername/syncclipboard-android/clipserver/internal/webdav"
)

type TestableClient interface {
	sync.SyncClient
	TestConnection() error
}

func New(account config.ServerAccount) (sync.SyncClient, error) {
	switch account.EffectiveType() {
	case config.ServerTypeWebDAV:
		return webdav.NewClient(account.URL, account.Username, account.Password)
	case config.ServerTypeOfficial, config.ServerTypeCustomOfficial:
		return serverclient.NewClient(account)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", account.EffectiveType())
	}
}

func NewTestable(account config.ServerAccount) (TestableClient, error) {
	switch account.EffectiveType() {
	case config.ServerTypeWebDAV:
		return webdav.NewClient(account.URL, account.Username, account.Password)
	case config.ServerTypeOfficial, config.ServerTypeCustomOfficial:
		return serverclient.NewClient(account)
	default:
		return nil, fmt.Errorf("unsupported server type: %s", account.EffectiveType())
	}
}
