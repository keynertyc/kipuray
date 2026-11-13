package browser

import (
	"context"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// command adapts a typed protocol call into a step without a returned value.
func command[P, R any](cmd cdp.Command[P, R], params P) chromedp.Action[chromedp.Void] {
	return chromedp.Func(func(ctx context.Context, target *chromedp.Target) error {
		_, err := cdp.Call(ctx, target, cmd, params)
		return err
	})
}
