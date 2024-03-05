package policy

import (
	"context"
	"testing"

	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestEngine_Match(t *testing.T) {
	list1 := &filterlist.StringRuleList{
		ID:             1,
		RulesText:      "||example.org\n",
		IgnoreCosmetic: false,
	}
	rules := []filterlist.RuleList{
		list1,
	}
	storage, err := filterlist.NewRuleStorage(rules)
	assert.NoError(t, err)
	e := urlfilter.NewNetworkEngine(storage)

	ctx := context.Background()
	engine := &Engine{
		ctx:    ctx,
		engine: e,
	}

	// Create a DNS message for testing
	msg := &dns.Msg{
		Question: []dns.Question{
			{
				Name:  "example.org.",
				Qtype: dns.TypeA,
			},
		},
	}

	result := engine.Match(msg)
	assert.True(t, result)
}
