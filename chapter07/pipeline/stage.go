package pipeline

import (
	"context"
	"golang.org/x/xerrors"
	"sync"
)

// FIFO dispatch strategy
type fifo struct {
	proc Processor
}

// FIFO returns a StageRunner that processes incoming payloads in a first-in
// first-out fashion. Each input is passed to the specified processor and its
// output is emitted to the next stage.
func FIFO(proc Processor) StageRunner {
	return fifo{proc: proc}
}

func (r fifo) Run(ctx context.Context, params StageParams) {
	for {
		select {
		case <-ctx.Done():
			// Asked to cleanly shut down
			return
		case payloadIn, ok := <-params.Input():
			if !ok {
				return
			}

			payloadOut, err := r.proc.Process(ctx, payloadIn)
			if err != nil {
				wrappedErr := xerrors.Errorf("pipeline stage %d: %w", params.StageIndex(), err)
				maybeEmitError(wrappedErr, params.Error())
				return
			}

			// If the processor did not output a payload for the
			// next stage there is nothing we need to do.
			if payloadOut == nil {
				payloadIn.MarkAsProcessed()
				continue
			}

			// Output processed data
			select {
			case params.Output() <- payloadOut:
			case <-ctx.Done():
				return
			}
		}
	}
}

// Fixed worker pool dispatch strategy
type fixedWorkerPool struct {
	fifos []StageRunner
}

func FixedWorkerPool(proc Processor, numWorkers int) StageRunner {
	if numWorkers <= 0 {
		panic("FixedWorkerPool: numWorkers must be > 0")
	}

	fifos := make([]StageRunner, numWorkers)
	for i := 0; i < numWorkers; i++ {
		fifos[i] = FIFO(proc)
	}

	// why using & operator
	return &fixedWorkerPool{fifos: fifos}
}

func (f fixedWorkerPool) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup

	for i, fifo := range f.fifos {
		wg.Add(1)
		go func(fifoIndex int) {
			fifo.Run(ctx, params)
		}(i)
	}

	wg.Wait()
}
