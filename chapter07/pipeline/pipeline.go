package pipeline

// maybeEmitError attempts to queue err to a buffered error channel. If the
// channel is full, the error is dropped.
func maybeEmitError(err error, errCh chan<- error) {
	select {
	case errCh <- err: // error emiited
	default: // error channel is full of other errors.
	}
}
