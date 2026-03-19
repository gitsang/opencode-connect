package plugin

import (
	"context"
	"fmt"
	"sync"
)

func Run(ctx context.Context, plugins []Plugin) error {
	if len(plugins) == 0 {
		return fmt.Errorf("no enabled plugins configured")
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(plugins))
	var waitGroup sync.WaitGroup

	for _, currentPlugin := range plugins {
		waitGroup.Add(1)
		go func(plugin Plugin) {
			defer waitGroup.Done()
			if err := plugin.Run(runCtx); err != nil {
				errCh <- fmt.Errorf("%s plugin failed: %w", plugin.Name(), err)
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
	case <-runCtx.Done():
		<-done
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	case err := <-errCh:
		<-done
		return err
	}
}
