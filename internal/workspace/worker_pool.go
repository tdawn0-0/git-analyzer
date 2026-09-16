package workspace

import (
	"context"
	"runtime"
	"sync"
)

// DefaultJobs returns min(NumCPU(), 8).
func DefaultJobs() int {
	n := runtime.NumCPU()
	if n > 8 {
		return 8
	}
	if n < 1 {
		return 1
	}
	return n
}

// Map runs fn over items with at most jobs workers. Results preserve input order.
// If ctx is cancelled, remaining work stops; completed results are still returned.
func Map[T any, R any](ctx context.Context, jobs int, items []T, fn func(context.Context, T) R) []R {
	if jobs <= 0 {
		jobs = DefaultJobs()
	}
	if jobs > len(items) && len(items) > 0 {
		jobs = len(items)
	}
	out := make([]R, len(items))
	if len(items) == 0 {
		return out
	}

	type job struct {
		i    int
		item T
	}
	ch := make(chan job)
	var wg sync.WaitGroup
	wg.Add(jobs)
	for w := 0; w < jobs; w++ {
		go func() {
			defer wg.Done()
			for j := range ch {
				if ctx.Err() != nil {
					continue
				}
				out[j.i] = fn(ctx, j.item)
			}
		}()
	}
	for i, item := range items {
		select {
		case <-ctx.Done():
			// stop sending; workers drain and exit
			goto done
		case ch <- job{i: i, item: item}:
		}
	}
done:
	close(ch)
	wg.Wait()
	return out
}
