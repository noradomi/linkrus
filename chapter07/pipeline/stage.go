package pipeline

import (
	"context"
	"golang.org/x/xerrors"
)

type fifo struct {
	proc Processor
}

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
