package testtasks

import (
	"context"
	"fmt"
	"time"
)

func PrintPayload(ctx context.Context, payload string) error {
	fmt.Println(payload)

	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
