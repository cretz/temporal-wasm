package host

import (
	"encoding/json"
	"fmt"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

type wasmWorkflowRun struct {
	info           *info
	infoJSON       []byte
	future         workflow.Future
	futureSet      workflow.Settable
	futureComplete bool
}

func newWASMWorkflowRun(ctx workflow.Context, info *info) (*wasmWorkflowRun, error) {
	r := &wasmWorkflowRun{info: info}
	var err error
	if r.infoJSON, err = json.Marshal(info); err != nil {
		return nil, err
	}
	r.future, r.futureSet = workflow.NewFuture(ctx)
	return r, nil
}

func (w *wasmWorkflowRun) wait(ctx workflow.Context) (*converter.RawPayloads, error) {
	var ret converter.RawPayloads
	if err := w.future.Get(ctx, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

// Safe for multiple calls, only first complete* taken.
func (w *wasmWorkflowRun) complete(b []byte) {
	if w.futureComplete {
		return
	}
	w.futureComplete = true
	// Try to convert to payloads
	var payloads []*payload
	if len(b) > 0 {
		if err := json.Unmarshal(b, &payloads); err != nil {
			w.futureSet.SetError(fmt.Errorf("failed unmarshalling completed payloads: %w", err))
			return
		}
	}
	w.futureSet.SetValue(payloadsToRaw(payloads))
}

// Safe for multiple calls, only first complete* taken.
func (w *wasmWorkflowRun) completeWithFailure(b []byte) {
	if w.futureComplete {
		return
	}
	w.futureComplete = true
	var f failure
	if err := json.Unmarshal(b, &f); err != nil {
		w.futureSet.SetError(fmt.Errorf("failed unmarshalling failure: %w", err))
		return
	}
	w.futureSet.SetError(f.toError())
}

// Safe for multiple calls, only first complete* taken.
func (w *wasmWorkflowRun) completeWithError(err error) {
	if w.futureComplete {
		return
	}
	w.futureComplete = true
	// TODO(cretz): Wrap as non-retryable?
	w.futureSet.SetError(err)
}
