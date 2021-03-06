package dkron

// Execution processor plugins must implement this interface
type ExecutionProcessor interface {
	// Main plugin method, will be called when an execution is done.
	Process(args *ExecutionProcessorArgs) Execution
}

// Arguments for calling an execution processor
type ExecutionProcessorArgs struct {
	// The execution to pass to the processor
	Execution Execution
	// The configuration for this plugin call
	Config PluginConfig
}

// Represents a plgin config data structure
type PluginConfig map[string]interface{}
