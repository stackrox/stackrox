package filewatcher

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	defaultMaxFileSize = 5 * 1024 * 1024 // 5 MB
	minInterval        = 5 * time.Second
)

var log = logging.LoggerForModule()

// Handler is called when the watched file's content changes.
// Return nil to acknowledge the content (hash is updated, same content
// will not be re-delivered). Return a non-nil error to request
// re-delivery on the next poll cycle.
type Handler func(data []byte) error

// Option configures a Watcher.
type Option func(*Watcher)

// WithMaxFileSize overrides the default maximum file size (5 MB).
func WithMaxFileSize(n int64) Option {
	return func(w *Watcher) { w.maxFileSize = n }
}

// WithOnError sets a callback invoked on file-level errors
// (stat failure, read failure, oversized file).
func WithOnError(fn func(error)) Option {
	return func(w *Watcher) { w.onError = fn }
}

// Watcher polls a single file for content changes using SHA-256 hashing.
type Watcher struct {
	filePath    string
	interval    time.Duration
	handler     Handler
	maxFileSize int64
	onError     func(error)
	stopSig     concurrency.Signal
	doneSig     concurrency.Signal
	lastHash    [sha256.Size]byte
}

// New creates a Watcher that polls filePath at the given interval.
// The handler is called with the raw file bytes whenever the content changes.
func New(filePath string, interval time.Duration, handler Handler, opts ...Option) *Watcher {
	if interval < minInterval {
		log.Warnf("Watch interval %v is below minimum %v, clamping", interval, minInterval)
		interval = minInterval
	}
	w := &Watcher{
		filePath:    filePath,
		interval:    interval,
		handler:     handler,
		maxFileSize: defaultMaxFileSize,
		stopSig:     concurrency.NewSignal(),
		doneSig:     concurrency.NewSignal(),
	}
	for _, o := range opts {
		o(w)
	}
	return w
}

// Start begins polling in a background goroutine.
func (w *Watcher) Start() {
	log.Infof("Starting file watcher for %q", w.filePath)
	go w.run()
}

// Stop signals the watcher to stop and blocks until it exits.
func (w *Watcher) Stop() {
	w.stopSig.Signal()
	<-w.doneSig.Done()
}

func (w *Watcher) run() {
	defer w.doneSig.Signal()

	w.check()

	t := time.NewTicker(w.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			w.check()
		case <-w.stopSig.Done():
			return
		}
	}
}

func (w *Watcher) check() {
	info, err := os.Stat(w.filePath)
	if errors.Is(err, os.ErrNotExist) {
		log.Debugf("Watched file %q does not exist, skipping", w.filePath)
		w.lastHash = [sha256.Size]byte{}
		return
	}
	if err != nil {
		log.Warnf("Failed to stat watched file %q: %v", w.filePath, err)
		w.reportError(fmt.Errorf("stat %q: %w", w.filePath, err))
		return
	}
	if info.Size() > w.maxFileSize {
		fingerprint := sha256.Sum256([]byte(fmt.Sprintf("oversize:%d:%d", info.Size(), info.ModTime().UnixNano())))
		if fingerprint != w.lastHash {
			log.Warnf("Watched file %q exceeds maximum size (%d bytes > %d), skipping",
				w.filePath, info.Size(), w.maxFileSize)
			w.lastHash = fingerprint
			w.reportError(fmt.Errorf("file %q exceeds maximum size (%d > %d)", w.filePath, info.Size(), w.maxFileSize))
		}
		return
	}

	data, err := os.ReadFile(w.filePath)
	if err != nil {
		log.Warnf("Failed to read watched file %q: %v", w.filePath, err)
		w.reportError(fmt.Errorf("read %q: %w", w.filePath, err))
		return
	}

	hash := sha256.Sum256(data)
	if hash == w.lastHash {
		log.Debugf("Watched file %q unchanged, skipping", w.filePath)
		return
	}

	if err := w.handler(data); err != nil {
		return
	}
	w.lastHash = hash
}

func (w *Watcher) reportError(err error) {
	if w.onError != nil {
		w.onError(err)
	}
}
