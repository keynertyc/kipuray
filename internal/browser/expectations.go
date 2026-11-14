package browser

import (
	"context"
	"time"

	"github.com/keynertyc/kipuray/internal/casefile"
)

func checkExpectation(ctx context.Context, e casefile.Expectation) Step {
	step := Step{Label: e.Kind + " " + e.Selector + " " + e.Value, Status: "failed", Detail: "Expected condition was not satisfied"}
	until := time.Now().Add(3 * time.Second)
	for {
		ok, err := evaluateExpectation(ctx, e)
		if err != nil {
			step.Status = "incomplete"
			step.Detail = err.Error()
			return step
		}
		if ok {
			step.Status = "passed"
			step.Detail = ""
			return step
		}
		if time.Now().After(until) {
			return step
		}
		select {
		case <-ctx.Done():
			step.Status = "incomplete"
			step.Detail = ctx.Err().Error()
			return step
		case <-time.After(50 * time.Millisecond):
		}
	}
}
