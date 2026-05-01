package uploader

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Job struct {
	Index   int
	Request AssetUploadRequest
}

type Result struct {
	Job       Job
	Operation Operation
	Asset     Asset
	Err       error
}

type UploadOptions struct {
	Concurrency int
	FailFast    bool
	Poll        PollOptions
}

func (client Client) UploadAsset(ctx context.Context, accessToken string, upload AssetUploadRequest, poll PollOptions) (Operation, Asset, error) {
	operation, err := client.CreateAsset(ctx, accessToken, upload)
	if err != nil {
		return Operation{}, Asset{}, err
	}

	asset, err := client.PollOperation(ctx, accessToken, operation, poll)
	if err != nil {
		return operation, Asset{}, err
	}
	return operation, asset, nil
}

func (client Client) UploadMany(ctx context.Context, accessToken string, jobs []Job, options UploadOptions) ([]Result, error) {
	if len(jobs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]Result, len(jobs))
	jobCh := make(chan Job)

	var wg sync.WaitGroup
	workerCount := options.Concurrency
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(jobs) {
		workerCount = len(jobs)
	}

	var firstErr error
	var firstErrMu sync.Mutex
	recordErr := func(err error) {
		if err == nil {
			return
		}
		firstErrMu.Lock()
		defer firstErrMu.Unlock()
		if firstErr == nil {
			firstErr = err
			if options.FailFast {
				cancel()
			}
		}
	}

	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				result := Result{Job: job}
				operation, asset, err := client.UploadAsset(ctx, accessToken, job.Request, options.Poll)
				result.Operation = operation
				result.Asset = asset
				result.Err = err
				results[job.Index] = result
				recordErr(err)
			}
		}()
	}

feed:
	for i, job := range jobs {
		if job.Index < 0 {
			job.Index = i
		}
		if job.Index >= len(jobs) {
			close(jobCh)
			wg.Wait()
			return results, fmt.Errorf("job index %d is outside result range", job.Index)
		}
		select {
		case <-ctx.Done():
			break feed
		case jobCh <- job:
		}
		if options.FailFast && ctx.Err() != nil {
			break feed
		}
	}
	close(jobCh)
	wg.Wait()

	firstErrMu.Lock()
	defer firstErrMu.Unlock()
	if firstErr != nil {
		return results, firstErr
	}
	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return results, err
	}
	return results, nil
}
