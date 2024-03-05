package policy

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

func TestLoadRemote(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write some data to the temporary file
	data := []byte("||example.org\n")
	if _, err := tmpfile.Write(data); err != nil {
		t.Fatalf("Failed to write data to temporary file: %v", err)
	}

	// Close the temporary file
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temporary file: %v", err)
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		if _, err := rw.Write(data); err != nil {
			t.Fatalf("Failed to write response data: %v", err)
		}
	}))
	defer server.Close()

	path := server.URL

	rand.New(rand.NewSource(time.Now().UnixNano()))
	tmpFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("coredns-%d.txt", rand.Int()))
	defer os.Remove(tmpFilePath)

	ctx, cancel := context.WithCancel(context.Background())
	engine := &Engine{
		ctx:         ctx,
		path:        path,
		tmpFilePath: tmpFilePath,
	}

	err = engine.loadRemote()
	assert.NoError(t, err)
	assert.NotNil(t, engine.engine)

	ok := engine.Match(&dns.Msg{
		Question: []dns.Question{
			{
				Name:  "example.org.",
				Qtype: dns.TypeA,
			},
		},
	})
	assert.True(t, ok)

	// test ctx canceled

	cancel()
	ok = engine.Match(&dns.Msg{
		Question: []dns.Question{
			{
				Name:  "example.org.",
				Qtype: dns.TypeA,
			},
		},
	})
	assert.False(t, ok)
}
