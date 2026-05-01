package testtasks

import (
	"fmt"
	"time"
)

func PrintPayload(args ...any) error {
	fmt.Printf("%v\n", args...)
	time.Sleep(1 * time.Second)
	return nil
}
