package provider

import (
	"fmt"
	"log"
	"time"
)

const (
	globalRetryCount = 30
	globalRetryTime  = 10 * time.Second
)

// RetryUntilTrue returns when there is an error or the requested operation returns true.
func RetryUntilTrue(name string, fn func() (bool, error)) error {
	for i := 1; i <= globalRetryCount; i++ {
		if ready, err := fn(); err != nil {
			return err
		} else if !ready {
			log.Printf("Request for '%v' not completed. Checking in %v", name, globalRetryTime)
			time.Sleep(globalRetryTime)
			continue
		}
		log.Printf("Request for '%v' is completed!", name)
		return nil
	}
	return fmt.Errorf("Request for '%v' not completed after retrying %d times", name, globalRetryCount)
}
