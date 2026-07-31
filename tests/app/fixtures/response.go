package fixtures

const (
	FeaturePanicMessage       = "feature panic with sensitive detail"
	HealthyResponseBody       = `{"status":"OK"}`
	InternalErrorResponseBody = `{"code":"IS","errors":{"description":"Internal server error."},"data":[]}`
)
