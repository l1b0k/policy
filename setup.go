package policy

import (
	"context"
	"time"

	"github.com/coredns/caddy"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("policy")

func init() { plugin.Register("policy", setup) }

func setup(c *caddy.Controller) error {
	ps, err := parsePolicy(c)
	if err != nil {
		return plugin.Error("policy", err)
	}
	for i := range ps {
		p := ps[i]
		dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
			p.Next = next
			return p
		})
		c.OnStartup(p.OnStartup)
		c.OnShutdown(p.OnShutdown)
	}
	return nil
}

func parsePolicy(c *caddy.Controller) ([]*Policy, error) {
	var fs []*Policy
	for c.Next() {
		f, err := parseStanza(c)
		if err != nil {
			return nil, err
		}

		fs = append(fs, f)
	}
	return fs, nil
}

func parseStanza(c *caddy.Controller) (*Policy, error) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	p := &Policy{
		ctx:    ctx,
		cancel: cancel,
	}

	if !c.Args(&p.rule) {
		return p, c.ArgErr()
	}

	for c.NextBlock() {
		err = parseBlock(c, p)
		if err != nil {
			return nil, err
		}
	}

	p.engine, err = newEngine(ctx, p.rule, p.cacheDir, p.period, p.base64)

	return p, err
}

func parseBlock(c *caddy.Controller, f *Policy) error {
	switch c.Val() {
	case "period":
		if c.NextArg() {
			period, err := time.ParseDuration(c.Val())
			if err != nil {
				return err
			}
			f.period = period
		}
	case "cache_dir":
		if c.NextArg() {
			f.cacheDir = c.Val()
		}
	case "base64":
		f.base64 = true
	}
	return nil
}
