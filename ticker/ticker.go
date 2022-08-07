package ticker

import (
	"context"
	"fmt"
	"time"
)

type TickerFunc struct {
	Logger interface{ Errorf(msg string, a ...any) }
	C      context.Context
	D      time.Duration
	F      func()
}

//DoTickerFunc is a very simple ticker function that, on Do does F in tickerFunc object until the context is called.
func (t TickerFunc) Do() error {
	if t.D <= time.Duration(0) {
		return fmt.Errorf("must have duration greater than 0")
	}

	if t.F == nil {
		return fmt.Errorf("function may not be nil")
	}

	if t.C == nil {
		return fmt.Errorf("function may not be nil")
	}

	ticker := time.NewTicker(t.D)

	go func() {
		defer func() {
			ticker.Stop()
			if e := recover(); e != nil {
				if t.Logger != nil {
					t.Logger.Errorf("Ran into exception when running ticker function: %+v", e)
				}
			}
		}()
		for {
			select {
			case <-ticker.C:
				t.F()
			case <-t.C.Done():
				return
			}
		}
	}()
	return nil
}

//SimpleTickerFunc takes in a duration and a function and creates a Ticker Function from those, returning a cancel
//so that the user can focus on sending a duration and function. The returned function when called will cancel the
//ticker. This function panics on values that would error in TickerFunc creation.

func SimpleTickerFunc(dur time.Duration, f func()) (cancel func()) {
	ctx, cancel := context.WithCancel(context.Background())

	//create the default ticker and start it.
	e := TickerFunc{
		C: ctx,
		D: dur,
		F: f,
	}.Do()
	if e != nil {
		panic(fmt.Errorf("failed to create simple ticker func: %w", e))
	}

	return cancel
}
