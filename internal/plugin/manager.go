package plugin

import (
	"context"
	"fmt"
	"sync"
)

func Run(ctx context.Context, plugins []Plugin, handle HandleFunc) error {
	if len(plugins) == 0 {
		return fmt.Errorf("no enabled plugins configured")
	}
	if handle == nil {
		return fmt.Errorf("handle function is required")
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(plugins))
	var waitGroup sync.WaitGroup

	for _, currentPlugin := range plugins {
		waitGroup.Add(1)
		go func(current Plugin) {
			defer waitGroup.Done()
			if err := current.Serve(runCtx, handle); err != nil {
				errCh <- fmt.Errorf("%s plugin failed: %w", current.Name(), err)
				cancel()
			}
		}(currentPlugin)
	}

	done := make(chan struct{})
	go func() {
		waitGroup.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		<-done
		return err
	case <-ctx.Done():
		<-done
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	case <-done:
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	}
}
