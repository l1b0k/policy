package policy

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

type EngineInterface interface {
	Match(r *dns.Msg) bool
	Run()
}

type Engine struct {
	ctx context.Context

	base64            bool
	path, tmpFilePath string // either be a local file or url

	storage *filterlist.RuleStorage
	engine  *urlfilter.NetworkEngine
	sync.RWMutex

	syncCh     chan struct{}
	syncPeriod time.Duration
}

func (e *Engine) Match(r *dns.Msg) bool {
	if e.ctx.Err() != nil {
		return false
	}

	if len(r.Question) == 0 {
		return false
	}

	e.RLock()
	defer e.RUnlock()

	req := rules.NewRequestForHostname(toHostName(r.Question[0].Name))
	_, ok := e.engine.Match(req)
	return ok
}

func (e *Engine) Run() {
	if e.syncPeriod == 0 {
		return
	}

	go func() {
		for {
			select {
			case <-e.ctx.Done():
				return
			case <-e.syncCh:
				err := e.reload()
				if err != nil {
					log.Infof("reload failed %v", err)
				}
			}
		}
	}()

	// create a ticker to reload rule
	go func() {
		c := time.NewTicker(e.syncPeriod)
		defer c.Stop()

		for {
			select {
			case <-c.C:
				e.syncCh <- struct{}{}
			case <-e.ctx.Done():
				return
			}
		}
	}()
}

func (e *Engine) reload() error {
	var err error

	if e.tmpFilePath == "" {
		err = e.reloadEngine(e.path)
	} else {
		err = e.loadRemote()
	}

	return err
}

func (e *Engine) reloadEngine(path string) error {
	storage, err := newStorageFromPath(path)
	if err != nil {
		return err
	}

	e.Lock()
	defer e.Unlock()

	if e.storage != nil {
		_ = e.storage.Close()
	}

	engine := urlfilter.NewNetworkEngine(storage)
	e.storage = storage
	e.engine = engine

	return nil
}

func (e *Engine) loadRemote() error {
	// 1. download and write to tmp file
	log.Infof("downloading rule %s", e.path)
	resp, err := http.Get(e.path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	log.Infof("download rule success, %s", e.path)

	err = os.MkdirAll(filepath.Dir(e.tmpFilePath), os.ModeDir)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(e.tmpFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	if e.base64 {
		log.Infof("decode b64")
		decoder := base64.NewDecoder(base64.StdEncoding, resp.Body)
		_, err = io.Copy(file, decoder)
	} else {
		_, err = io.Copy(file, resp.Body)
	}

	if err != nil {
		_ = file.Close()
		return err
	}
	_ = file.Close()

	log.Infof("write to %s", file.Name())
	return e.reloadEngine(e.tmpFilePath)
}

// newEngine create a new rule engine.
// path is for the rule file url or local file path.
// cacheDir is used to store the downloaded file. If unset , the tmp file will be created in the tmp dir. We will NOT delete it.
func newEngine(ctx context.Context, path, cacheDir string, syncPeriod time.Duration, base64 bool) (*Engine, error) {
	var tmpFilePath string
	if isUrl(path) {
		if cacheDir != "" {
			tmpFilePath = filepath.Join(cacheDir, filepath.Base(path))
		} else {
			rand.New(rand.NewSource(time.Now().UnixNano()))
			tmpFilePath = filepath.Join(os.TempDir(), fmt.Sprintf("coredns-%d.txt", rand.Int()))
		}
	}
	e := &Engine{ctx: ctx, path: path, syncCh: make(chan struct{}), syncPeriod: syncPeriod, tmpFilePath: tmpFilePath, base64: base64}

	// not necessary to download at first place
	_, err := os.Stat(tmpFilePath)
	if err != nil {
		err = e.reload()
	} else {
		err = e.reloadEngine(tmpFilePath)
	}

	log.Infof("new engine, path %s sync period %s", path, syncPeriod.String())
	return e, err
}

func newStorageFromPath(path string) (*filterlist.RuleStorage, error) {
	var ruleLists []filterlist.RuleList
	ruleList, err := filterlist.NewFileRuleList(0, path, false)
	if err != nil {
		return nil, err
	}
	ruleLists = append(ruleLists, ruleList)
	if err != nil {
		return nil, err
	}

	return filterlist.NewRuleStorage(ruleLists)
}

func isUrl(path string) bool {
	u, err := url.Parse(path)
	if err == nil && u.Scheme != "" {
		return true
	}
	return false
}
