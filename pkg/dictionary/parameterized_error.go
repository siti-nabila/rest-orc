package dictionary

import (
	_ "embed"
	"time"
)

var (
	//go:embed parameterized_err_list.yaml
	parameterizedErrorList []byte

	parameterizedErrorPack = mustLoadErrorDictionary(parameterizedErrorList)

	ErrMinimumDurationWhenEnabled = parameterizedErrorPack.New(
		"minimum_duration_when_enabled",
	)
	ErrInvalidConfigurationField          = parameterizedErrorPack.New("invalid_configuration_field")
	ErrLoadApplicationConfiguration       = parameterizedErrorPack.New("load_application_configuration")
	ErrCreateApplication                  = parameterizedErrorPack.New("create_application")
	ErrListenHTTPServer                   = parameterizedErrorPack.New("listen_http_server")
	ErrShutdownHTTPServer                 = parameterizedErrorPack.New("shutdown_http_server")
	ErrCloseAuthGRPCConnection            = parameterizedErrorPack.New("close_auth_grpc_connection")
	ErrCloseApplication                   = parameterizedErrorPack.New("close_application")
	ErrOpenConfiguration                  = parameterizedErrorPack.New("open_configuration")
	ErrDecodeConfiguration                = parameterizedErrorPack.New("decode_configuration")
	ErrCloseConfiguration                 = parameterizedErrorPack.New("close_configuration")
	ErrParseEnvironmentVariable           = parameterizedErrorPack.New("parse_environment_variable")
	ErrDecodeDuration                     = parameterizedErrorPack.New("decode_duration")
	ErrParseDuration                      = parameterizedErrorPack.New("parse_duration")
	ErrConfigureHTTPClientTLS             = parameterizedErrorPack.New("configure_http_client_tls")
	ErrParseHTTPClientBaseURL             = parameterizedErrorPack.New("parse_http_client_base_url")
	ErrAbsoluteHTTPClientBaseURLRequired  = parameterizedErrorPack.New("absolute_http_client_base_url_required")
	ErrUnsupportedHTTPClientBaseURLScheme = parameterizedErrorPack.New("unsupported_http_client_base_url_scheme")
	ErrHTTPClientTLSSchemeMismatch        = parameterizedErrorPack.New("http_client_tls_scheme_mismatch")
	ErrExecuteHTTPRequest                 = parameterizedErrorPack.New("execute_http_request")
	ErrCreateGRPCClient                   = parameterizedErrorPack.New("create_grpc_client")
	ErrCloseGRPCClient                    = parameterizedErrorPack.New("close_grpc_client")
	ErrConfigureGRPCClientTLS             = parameterizedErrorPack.New("configure_grpc_client_tls")
	ErrReadTLSCAFile                      = parameterizedErrorPack.New("read_tls_ca_file")
	ErrLoadSystemCertificatePool          = parameterizedErrorPack.New("load_system_certificate_pool")
	ErrTLSCACertificatesRequired          = parameterizedErrorPack.New("tls_ca_certificates_required")
)

func MinimumDurationWhenEnabled(minimum time.Duration) error {
	return parameterizedErrorPack.Newf("minimum_duration_when_enabled", minimum)
}

func LoadApplicationConfiguration(cause error) error {
	return newErrorWithCause("load_application_configuration", cause, cause)
}

func CreateApplication(cause error) error {
	return newErrorWithCause("create_application", cause, cause)
}

func ListenHTTPServer(address string, cause error) error {
	return newErrorWithCause("listen_http_server", cause, address, cause)
}

func ShutdownHTTPServer(cause error) error {
	return newErrorWithCause("shutdown_http_server", cause, cause)
}

func CloseAuthGRPCConnection(cause error) error {
	return newErrorWithCause("close_auth_grpc_connection", cause, cause)
}

func CloseApplication(cause error) error {
	return newErrorWithCause("close_application", cause, cause)
}

func OpenConfiguration(path string, cause error) error {
	return newErrorWithCause("open_configuration", cause, path, cause)
}

func DecodeConfiguration(path string, cause error) error {
	return newErrorWithCause("decode_configuration", cause, path, cause)
}

func CloseConfiguration(path string, cause error) error {
	return newErrorWithCause("close_configuration", cause, path, cause)
}

func ParseEnvironmentVariable(key, value string, cause error) error {
	return newErrorWithCause("parse_environment_variable", cause, key, value, cause)
}

func DecodeDuration(cause error) error {
	return newErrorWithCause("decode_duration", cause, cause)
}

func ParseDuration(value string, cause error) error {
	return newErrorWithCause("parse_duration", cause, value, cause)
}

func ConfigureHTTPClientTLS(cause error) error {
	return newErrorWithCause("configure_http_client_tls", cause, cause)
}

func ParseHTTPClientBaseURL(baseURL string, cause error) error {
	return newErrorWithCause("parse_http_client_base_url", cause, baseURL, cause)
}

func AbsoluteHTTPClientBaseURLRequired(baseURL string) error {
	return parameterizedErrorPack.Newf("absolute_http_client_base_url_required", baseURL)
}

func UnsupportedHTTPClientBaseURLScheme(baseURL string) error {
	return parameterizedErrorPack.Newf("unsupported_http_client_base_url_scheme", baseURL)
}

func HTTPClientTLSSchemeMismatch(scheme string) error {
	return parameterizedErrorPack.Newf("http_client_tls_scheme_mismatch", scheme)
}

func ExecuteHTTPRequest(cause error) error {
	return newErrorWithCause("execute_http_request", cause, cause)
}

func CreateGRPCClient(target string, cause error) error {
	return newErrorWithCause("create_grpc_client", cause, target, cause)
}

func CloseGRPCClient(cause error) error {
	return newErrorWithCause("close_grpc_client", cause, cause)
}

func ConfigureGRPCClientTLS(cause error) error {
	return newErrorWithCause("configure_grpc_client_tls", cause, cause)
}

func ReadTLSCAFile(path string, cause error) error {
	return newErrorWithCause("read_tls_ca_file", cause, path, cause)
}

func LoadSystemCertificatePool(cause error) error {
	return newErrorWithCause("load_system_certificate_pool", cause, cause)
}

func TLSCACertificatesRequired(path string) error {
	return parameterizedErrorPack.Newf("tls_ca_certificates_required", path)
}
