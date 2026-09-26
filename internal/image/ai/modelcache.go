package ai

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// modelCache caches a provider's most recent model-availability result for a TTL.
type modelCache struct {
	mu           sync.Mutex
	providerName string
	ttl          time.Duration
	checkedAt    time.Time
	available    bool
	valid        bool
}

// newModelCache creates a new modelCache for the given provider, caching results for ttl.
// providerName: A human-readable label used in debug log messages.
// ttl: How long a cached result remains valid before the underlying check is repeated.
func newModelCache(providerName string, ttl time.Duration) *modelCache {
	return &modelCache{providerName: providerName, ttl: ttl}
}

// checkModelCached returns the cached availability result if it is still within the TTL.
// Otherwise it calls check to perform the real lookup and caches the result for subsequent calls.
// check: The underlying model-availability check to perform on a cache miss.
func (c *modelCache) checkModelCached(check func() bool) bool {
	c.mu.Lock()
	if c.valid {
		age := time.Since(c.checkedAt)
		if age < c.ttl {
			available := c.available
			c.mu.Unlock()
			logrus.Debugf(
				"using cached %s model availability: %v (checked %s ago)",
				c.providerName,
				available,
				age,
			)
			return available
		}
	}
	c.mu.Unlock()

	available := check()

	c.mu.Lock()
	c.available = available
	c.checkedAt = time.Now()
	c.valid = true
	c.mu.Unlock()

	return available
}
