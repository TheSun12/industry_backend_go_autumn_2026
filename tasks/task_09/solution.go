package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}

	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	if len(in) == 0 {
		return []Result[R]{}, nil
	}

	results := make([]Result[R], len(in))

	var wg sync.WaitGroup
	var mu sync.Mutex

	count := 0

	count_workers := workers
	if count_workers > len(in) {
		count_workers = len(in)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(count_workers)

	for i := 0; i < count_workers; i++ {
		go func() {
			defer wg.Done()
			for {
				mu.Lock()

				if ctx.Err() != nil || count >= len(in) {
					mu.Unlock()
					return
				}
				j := count
				count++
				mu.Unlock()

				val, err := fn(ctx, in[j])
				if err != nil {
					results[j] = Result[R]{Err: err}
				} else {
					results[j] = Result[R]{Value: val}
				}
			}
		}()
	}

	wg.Wait()
	
	error_ := ctx.Err()
	if error_ != nil {
		return nil, error_
	}

	return results, nil
}
