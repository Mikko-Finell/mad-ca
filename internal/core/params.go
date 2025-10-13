package core

// ParamType enumerates supported parameter value kinds.
type ParamType string

const (
	// ParamTypeInt denotes integer-valued parameters.
	ParamTypeInt ParamType = "int"
	// ParamTypeFloat denotes floating-point parameters.
	ParamTypeFloat ParamType = "float"
	// ParamTypeBool denotes boolean parameters.
	ParamTypeBool ParamType = "bool"
)

// Parameter describes a single tunable value exposed by a simulation.
type Parameter struct {
	Key         string
	Label       string
	Type        ParamType
	Value       string
	Description string
}

// ParameterGroup clusters related parameters for presentation purposes.
type ParameterGroup struct {
	Name    string
	Params  []Parameter
	Summary string
}

// ParameterSnapshot captures the current set of tunables exposed by a sim.
type ParameterSnapshot struct {
	Groups []ParameterGroup
}
