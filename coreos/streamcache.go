package coreos

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/stream-metadata-go/fedoracoreos"
	"github.com/coreos/stream-metadata-go/stream"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// StreamCache maintains a local copy of the Stream JSON info
// fetched from Fedora.
type StreamCache struct {
	LocalDir string
	m        map[string]*stream.Stream
	mu       sync.Mutex
}

func (c *StreamCache) init() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[string]*stream.Stream)
	}
}

func (c *StreamCache) LoadAll() error {
	for _, name := range []string{fedoracoreos.StreamStable, fedoracoreos.StreamTesting, fedoracoreos.StreamNext} {
		if _, err := c.Get(name); err != nil {
			return err
		}
	}
	return nil
}

func (c *StreamCache) Put(s *stream.Stream) {
	c.init()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[s.Stream] = s
}

func (c *StreamCache) Get(name string) (*stream.Stream, error) {
	c.init()
	c.mu.Lock()
	defer c.mu.Unlock()

	if s, ok := c.m[name]; ok {
		log.Printf("CoreOS Stream[%s] Read from memory", name)
		return s, nil
	}
	s, err := c.readFile(name)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		log.Printf("CoreOS Stream[%s] Read from File", name)
		c.m[name] = s
		return s, nil
	}
	log.Printf("CoreOS Stream[%s] Fetch from URL", name)
	s, err = fedoracoreos.FetchStream(name)
	if err != nil {
		return nil, fmt.Errorf("rrror fetching stream %s: %w", name, err)
	}
	if err := c.writeFile(s); err != nil {
		log.Printf("Error writing stream: %s", err)
	}
	c.m[name] = s
	return s, nil
}

func (c *StreamCache) readFile(name string) (*stream.Stream, error) {
	localFile := filepath.Join(c.LocalDir, name+".json")
	body, err := os.ReadFile(localFile)
	if err != nil {
		return nil, err
	}

	var s stream.Stream
	err = json.Unmarshal(body, &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *StreamCache) writeFile(s *stream.Stream) error {
	body, err := json.Marshal(s)
	if err != nil {
		return err
	}
	localFile := filepath.Join(c.LocalDir, s.Stream+".json")
	return os.WriteFile(localFile, body, 0664)
}
