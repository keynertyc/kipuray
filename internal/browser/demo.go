package browser

import (
	"context"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/keynertyc/kipuray/internal/casefile"
)

// RecordDemo creates genuine browser evidence for the bundled synthetic checkout.
// It uses the same capture path as a manual recording, with one scripted click.
func RecordDemo(ctx context.Context, target string, options Options) (*casefile.Case, error) {
	options.Headless = true
	capture, err := StartCapture(ctx, target, CaptureOptions{Options: options, CaptureJSON: true})
	if err != nil {
		return nil, err
	}
	stopped := false
	defer func() {
		if !stopped {
			capture.Stop()
		}
	}()
	if err := chromedp.Do(capture.ctx, chromedp.WaitVisible(chromedp.CSS(`[data-testid="apply"]`)), chromedp.Click(chromedp.CSS(`[data-testid="apply"]`))); err != nil {
		return nil, err
	}
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		if len(capture.Snapshot().Fixtures) > 0 {
			break
		}
		select {
		case <-time.After(30 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	// Let rrweb receive the DOM/error events associated with the completed request.
	select {
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	c := capture.Stop()
	stopped = true
	c.Title = "Checkout · null discount regression"
	c.Expectations = []casefile.Expectation{{Kind: "text", Selector: `[data-testid="message"]`, Value: "Coupon not valid"}}
	return c, nil
}
