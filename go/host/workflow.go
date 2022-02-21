package host

import (
	"fmt"
	"os"

	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

type wasmWorkflow struct {
	// Will be nil if from-first-workflow-param
	module         module
	fromFirstParam bool
	engine         engine
}

type Option func(*wasmWorkflow) error

type engine interface {
	newModule([]byte) (module, error)
}

type module interface {
	newInstance(*wasmWorkflowRun) (instance, error)
}

type instance interface {
	run() error
}

var defaultEngine engine = wazeroEngine{}

func NewWASMWorkflow(opts ...Option) (interface{}, error) {
	w := &wasmWorkflow{engine: defaultEngine}
	for _, opt := range opts {
		if err := opt(w); err != nil {
			return nil, err
		}
	}
	if w.module == nil && !w.fromFirstParam {
		return nil, fmt.Errorf("WASM option not set")
	}
	return w.WASMWorkflow, nil
}

func WASMFromFile(wasmFile string) Option {
	return func(w *wasmWorkflow) error {
		if w.module != nil || w.fromFirstParam {
			return fmt.Errorf("WASM option already set")
		}
		b, err := os.ReadFile(wasmFile)
		if err != nil {
			return fmt.Errorf("failed reading %v: %w", wasmFile, err)
		}
		w.module, err = w.engine.newModule(b)
		return err
	}
}

func WASMFromBytes(b []byte) Option {
	return func(w *wasmWorkflow) (err error) {
		if w.module != nil || w.fromFirstParam {
			return fmt.Errorf("WASM option already set")
		}
		w.module, err = w.engine.newModule(b)
		return err
	}
}

func WASMFromFirstWorkflowParam() Option {
	return func(w *wasmWorkflow) (err error) {
		if w.module != nil || w.fromFirstParam {
			return fmt.Errorf("WASM option already set")
		}
		w.fromFirstParam = true
		return nil
	}
}

// This is named as such for those using the default function name during
// registration
func (w *wasmWorkflow) WASMWorkflow(
	ctx workflow.Context,
	params *converter.RawPayloads,
) (*converter.RawPayloads, error) {
	// Build run
	info := &info{Params: payloadsFromRaw(params)}
	run, err := newWASMWorkflowRun(ctx, info)
	if err != nil {
		return nil, fmt.Errorf("failed creating run: %w", err)
	}

	// Build module
	module := w.module
	if module == nil {
		// Take from first param
		bytesFirstParam := len(info.Params) > 0 &&
			string(info.Params[0].Metadata[converter.MetadataEncoding]) == converter.MetadataEncodingBinary
		if !bytesFirstParam {
			return nil, fmt.Errorf("expected WASM as the first parameter")
		}
		var err error
		if module, err = w.engine.newModule(info.Params[0].Data); err != nil {
			return nil, fmt.Errorf("failed creating WASM module")
		}
		// Shift off the first param
		info.Params = info.Params[1:]
	}

	// Create instance
	instance, err := module.newInstance(run)
	if err != nil {
		return nil, fmt.Errorf("failed creating instance: %w", err)
	}

	// Run in background, setting any potential error
	workflow.Go(ctx, func(workflow.Context) {
		if err := instance.run(); err != nil {
			run.completeWithError(fmt.Errorf("failed running instance: %w", err))
		} else {
			run.completeWithError(fmt.Errorf("no completion value given: %w", err))
		}
	})

	// Wait for run to complete
	return run.wait(ctx)
}
