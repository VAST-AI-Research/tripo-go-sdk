package tripo3d

// Bool, Int64, Float64, and String return a pointer to the given value, for
// conveniently filling the optional (pointer-typed) fields of the various
// Params structs, e.g.:
//
//	tripo3d.TextToModelParams{
//	    Prompt:  "a cute cat",
//	    Texture: tripo3d.Bool(true),
//	}
func Bool(v bool) *bool          { return &v }
func Int64(v int64) *int64       { return &v }
func Float64(v float64) *float64 { return &v }
func String(v string) *string    { return &v }
