package fixtures

const (
	FeaturePanicMessage       = "feature panic with sensitive detail"
	HealthyResponseBody       = `{"status":"OK"}`
	InternalErrorResponseBody = `{"code":"IS","errors":{"description":"Internal server error."},"data":[]}`
	EndpointNotFoundBody      = `{"code":"NF","errors":{"description":"Endpoint not found."},"data":[]}`
	HTTPListenMessage         = "[REST-ORC] is running at http://0.0.0.0:8080 (port 8080)"
	HTTPSListenMessage        = "[REST-ORC] is running at https://0.0.0.0:8443 (port 8443)"
	AuthGRPCClientMessage     = "[CLIENT]: auth_grpc target: localhost:9090 (port 9090)"
	ResolverClientMessage     = "[CLIENT]: auth_grpc target: dns:///auth-service:50051 (port 50051)"
	PortlessClientMessage     = "[CLIENT]: auth_grpc target: unix:///tmp/auth.sock"
)
