package policy

import (
	"context"
	"strings"
	"time"

	"github.com/coredns/coredns/request"

	"github.com/coredns/coredns/plugin"

	"github.com/miekg/dns"
)

// Policy is a plugin that returns a HINFO reply to ANY queries.
type Policy struct {
	Next plugin.Handler

	ctx    context.Context
	cancel context.CancelFunc

	rule string // rule file either be the path of a file or a URL

	period time.Duration
	base64 bool

	cacheDir string

	engine EngineInterface
}

func (a *Policy) Filter(ctx context.Context, req *request.Request) bool {
	state := request.Request{W: req.W, Req: req.Req}
	switch state.QType() {
	case dns.TypeA, dns.TypeAAAA:
	default:
		return false
	}
	ok := a.engine.Match(req.Req)
	log.Debugf("filter %s %v", a.rule, ok)
	return ok
}

func (a *Policy) ViewName() string {
	return "policy/" + a.rule
}

// ServeDNS implements the plugin.Handler interface.
func (a *Policy) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	return plugin.NextOrFailure(a.Name(), a.Next, ctx, w, r)
}

// Name implements the Handler interface.
func (a *Policy) Name() string { return "policy" }

func (a *Policy) OnStartup() error {
	a.engine.Run()
	return nil
}

func (a *Policy) OnShutdown() error {
	a.cancel()
	return nil
}

func toHostName(in string) string {
	return strings.TrimSuffix(in, ".")
}
