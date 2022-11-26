package shim

import (
	"bytes"
	"context"
	"io"
	"os"
	"time"

	"github.com/stackrox/rox/central/authprovider/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// shimAuthProvider is a wrapper around both an underlying store (which is either postgres, bolt, rocksdb) and reading
// values from config maps mounted in a specific central directory.
type shimAuthProvider struct {
	underlying       store.Store
	authProviderLock sync.RWMutex
	providers        map[string]*storage.AuthProvider
}

var (
	_   store.Store = (*shimAuthProvider)(nil)
	log             = logging.LoggerForModule()
)

const (
	authProviderPath = "/run/config/stackrox.io/declarative-config/auth-providers/"
	watchInterval    = 5 * time.Second
)

// New provides a shim for the auth provider store, which will consult besides the underlying store
// a config map mount consisting of declarative objects created via e.g. custom resources in kubernetes.
// TODO(dhaus): right now skipping SAC checks for simplicity, they should be added as well.
// TODO(dhaus): we do not necessarily do anything non-generic here, so this could be covered by code generation (which will make SAC parts easier as well).
func New(underlying store.Store) store.Store {
	watchOpts := k8scfgwatch.Options{
		Interval: watchInterval,
		Force:    true,
	}

	shim := &shimAuthProvider{
		underlying: underlying,
	}

	wh := &watchHandler{shim: shim}
	_ = k8scfgwatch.WatchConfigMountDir(context.Background(), authProviderPath, k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)

	return shim
}

func (s *shimAuthProvider) GetAll(ctx context.Context) ([]*storage.AuthProvider, error) {
	providers, err := s.underlying.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	s.authProviderLock.RLock()
	defer s.authProviderLock.RUnlock()
	for _, provider := range s.providers {
		providers = append(providers, provider)
	}
	return providers, nil
}

func (s *shimAuthProvider) Get(ctx context.Context, id string) (*storage.AuthProvider, bool, error) {
	s.authProviderLock.RLock()
	defer s.authProviderLock.RUnlock()
	if provider, ok := s.providers[id]; ok {
		return provider, true, nil
	}

	return s.underlying.Get(ctx, id)
}

func (s *shimAuthProvider) Exists(ctx context.Context, id string) (bool, error) {
	s.authProviderLock.RLock()
	defer s.authProviderLock.RUnlock()
	if _, ok := s.providers[id]; ok {
		return true, nil
	}

	return s.underlying.Exists(ctx, id)
}

func (s *shimAuthProvider) Upsert(ctx context.Context, obj *storage.AuthProvider) error {
	s.authProviderLock.RLock()
	defer s.authProviderLock.RUnlock()

	// Special case for auth provider: The ID and name is equal, hence we can simply check it here.
	if _, ok := s.providers[obj.GetName()]; ok {
		return errox.InvalidArgs.Newf("auth provider with name %s already exists", obj.GetName())
	}

	return s.underlying.Upsert(ctx, obj)
}

func (s *shimAuthProvider) Delete(ctx context.Context, id string) error {
	s.authProviderLock.RLock()
	defer s.authProviderLock.RUnlock()

	if _, ok := s.providers[id]; ok {
		return errox.InvalidArgs.Newf("auth provider with id %s cannot be deleted via API", id)
	}

	return s.underlying.Delete(ctx, id)
}

type watchHandler struct {
	shim *shimAuthProvider
}

func (h *watchHandler) OnChange(dir string) (interface{}, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var openFiles []*os.File
	for _, file := range files {
		f, err := os.Open(file.Name())
		if err != nil {
			return nil, err
		}
		openFiles = append(openFiles, f)
	}
	defer closeAllFiles(openFiles)

	authProviders := make([]*storage.AuthProvider, 0, len(openFiles))
	for _, f := range openFiles {
		provider, err := readProtoFile(f)
		if err != nil {
			return nil, err
		}
		authProviders = append(authProviders, provider)
	}

	log.Infof("Got the following auth providers from the config map watch: %+v", authProviders)

	return authProviders, nil
}

func (h *watchHandler) OnStableUpdate(val interface{}, err error) {
	if err != nil {
		log.Warn("Error reading the auth provider config map files. This may lead to updates not being made to them.")
		return
	}

	authProviders, _ := val.([]*storage.AuthProvider)
	newAuthProviderMap := make(map[string]*storage.AuthProvider, len(authProviders))
	for _, provider := range authProviders {
		newAuthProviderMap[provider.GetId()] = provider
	}

	h.shim.authProviderLock.Lock()
	defer h.shim.authProviderLock.Unlock()
	h.shim.providers = newAuthProviderMap
	log.Infof("After updating, the auth provider map will hold the following entries: %+v", newAuthProviderMap)
}

func (h *watchHandler) OnWatchError(err error) {
	log.Errorf("Error watching for auth provider file changes: %v", err)
}

func closeAllFiles(files []*os.File) {
	for _, file := range files {
		utils.IgnoreError(file.Close)
	}
}

func readProtoFile(file io.Reader) (*storage.AuthProvider, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}

	authProvider := &storage.AuthProvider{}
	if err := authProvider.Unmarshal(buf.Bytes()); err != nil {
		return nil, err
	}

	return authProvider, nil
}
