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

// ParameterControl describes an adjustable parameter that should be exposed on
// the HUD. Steps and bounds are optional and interpreted based on the
// parameter type.
type ParameterControl struct {
	Key   string
	Label string
	Type  ParamType

	Step float64

	Min    float64
	Max    float64
	HasMin bool
	HasMax bool
}

// ParameterControlsProvider exposes the list of HUD-adjustable controls.
type ParameterControlsProvider interface {
	ParameterControls() []ParameterControl
}

// IntParameterSetter allows HUD interactions to update integer parameters.
type IntParameterSetter interface {
	SetIntParameter(key string, value int) bool
}

// FloatParameterSetter allows HUD interactions to update floating point
// parameters.
type FloatParameterSetter interface {
	SetFloatParameter(key string, value float64) bool
}
