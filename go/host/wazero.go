package host

import (
	"fmt"
	"log"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/wasm"
)

type wazeroEngine struct{}

func (wazeroEngine) newModule(b []byte) (module, error) {
	// Create the module
	mod := &wazero.ModuleConfig{Source: b}
	if err := mod.Validate(); err != nil {
		return nil, fmt.Errorf("failed decoding WASM: %w", err)
	}
	return &wazeroModule{mod}, nil
}

type wazeroModule struct{ mod *wazero.ModuleConfig }

func (w *wazeroModule) newInstance(run *wasmWorkflowRun) (instance, error) {
	// Create a store
	store := wazero.NewStoreWithConfig(&wazero.StoreConfig{Engine: wazero.NewEngineJIT()})

	// Bind the functions
	_, err := wazero.InstantiateHostModule(store, &wazero.HostModuleConfig{
		Name: "env",
		Functions: map[string]interface{}{
			"complete": func(ctx wasm.ModuleContext, offset, count uint32) {
				if b, ok := w.readMem(ctx, run, offset, count); ok {
					run.complete(b)
				}
			},
			"complete_with_failure": func(ctx wasm.ModuleContext, offset, count uint32) {
				if b, ok := w.readMem(ctx, run, offset, count); ok {
					run.completeWithFailure(b)
				}
			},
			"get_info": func(ctx wasm.ModuleContext, offset, count uint32) {
				if count != uint32(len(run.infoJSON)) {
					run.completeWithError(fmt.Errorf("invalid info length"))
					return
				}
				w.writeMem(ctx, run, offset, run.infoJSON)
			},
			"get_info_len": func(ctx wasm.ModuleContext) uint32 {
				return uint32(len(run.infoJSON))
			},
			"write_log": func(ctx wasm.ModuleContext, level, offset, count uint32) {
				var levelStr string
				switch level {
				case 1:
					levelStr = "ERROR"
				case 2:
					levelStr = " WARN"
				case 3:
					levelStr = " INFO"
				case 4:
					levelStr = "DEBUG"
				case 5:
					levelStr = "TRACE"
				default:
					return
				}
				if b, ok := w.readMem(ctx, run, offset, count); ok {
					log.Print("[" + levelStr + "] " + string(b))
				}
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed exporting functions: %w", err)
	}

	// Instantiate and start "run"
	exports, err := wazero.InstantiateModule(store, w.mod)
	if err != nil {
		return nil, fmt.Errorf("failed instantiating module: %w", err)
	}
	runFn := exports.Function("run")
	if runFn == nil {
		return nil, fmt.Errorf("missing 'run' function")
	}
	return &wazeroInstance{runFn: runFn}, nil
}

type wazeroInstance struct{ runFn wasm.Function }

func (w *wazeroInstance) run() error {
	_, err := w.runFn.Call(nil)
	return err
}

func (*wazeroModule) readMem(ctx wasm.ModuleContext, run *wasmWorkflowRun, offset, count uint32) ([]byte, bool) {
	b, ok := ctx.Memory().Read(offset, count)
	if !ok {
		run.completeWithError(fmt.Errorf("failed reading memory"))
	}
	return b, ok
}

func (*wazeroModule) writeMem(ctx wasm.ModuleContext, run *wasmWorkflowRun, offset uint32, b []byte) {
	if !ctx.Memory().Write(offset, b) {
		run.completeWithError(fmt.Errorf("failed writing memory"))
	}
}
