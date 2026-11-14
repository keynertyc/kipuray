package browser

import (
	"context"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
)

const networkQuietPeriod = 200 * time.Millisecond

// networkActivity follows the browser's complete request lifecycle, not just
// the time spent fulfilling a paused request. CDP may acknowledge a fulfillment
// before JavaScript has received and processed its response.
type networkActivity struct {
	mu            sync.Mutex
	requests      map[network.RequestID]struct{}
	interceptions int
	updated       time.Time
}

func newNetworkActivity() *networkActivity {
	return &networkActivity{requests: map[network.RequestID]struct{}{}, updated: time.Now()}
}
func (a *networkActivity) start(id network.RequestID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests[id] = struct{}{}
	a.updated = time.Now()
}
func (a *networkActivity) finish(id network.RequestID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.requests, id)
	a.updated = time.Now()
}
func (a *networkActivity) interceptStart() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.interceptions++
	a.updated = time.Now()
}
func (a *networkActivity) interceptFinish() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.interceptions--
	a.updated = time.Now()
}
func (a *networkActivity) touch() { a.mu.Lock(); defer a.mu.Unlock(); a.updated = time.Now() }

func (a *networkActivity) settle(ctx context.Context) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		a.mu.Lock()
		quiet := len(a.requests) == 0 && a.interceptions == 0 && time.Since(a.updated) >= networkQuietPeriod
		a.mu.Unlock()
		if quiet {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
