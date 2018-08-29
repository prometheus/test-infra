package provider

import (
	"fmt"
	"log"
	"time"
)

const (
	GlobalRetryCount = 30
	globalRetryTime  = 10 * time.Second
)

//common struct used by k8s.go & gke.go to save name and content of input files
type ResourceFile struct {
	Name    string
	Content []byte
}

// RetryUntilTrue returns when there is an error or the requested operation returns true.
func RetryUntilTrue(name string, retryCount int, fn func() (bool, error)) error {
	for i := 1; i <= retryCount; i++ {
		if ready, err := fn(); err != nil {
			return err
		} else if !ready {
			log.Printf("Request for '%v' is in progress. Checking in %v", name, globalRetryTime)
			time.Sleep(globalRetryTime)
			continue
		}
		log.Printf("Request for '%v' is done!", name)
		return nil
	}
	return fmt.Errorf("Request for '%v' hasn't completed after retrying %d times", name, retryCount)
}
