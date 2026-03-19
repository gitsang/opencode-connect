package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const defaultStopTimeout = 15 * time.Second

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
			if err := plugin.Start(runCtx); err != nil {
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

	var runErr error
	select {
	case err := <-errCh:
		runErr = err
		cancel()
	case <-ctx.Done():
		cancel()
	case <-done:
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), defaultStopTimeout)
	defer stopCancel()

	stopErr := stopAll(stopCtx, plugins)
	<-done

	if runErr != nil {
		return runErr
	}

	return stopErr
}

func stopAll(ctx context.Context, plugins []Plugin) error {
	var firstErr error
	for index := len(plugins) - 1; index >= 0; index-- {
		currentPlugin := plugins[index]
		if err := currentPlugin.Stop(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stop %s plugin: %w", currentPlugin.Name(), err)
		}
	}

	return firstErr
}
