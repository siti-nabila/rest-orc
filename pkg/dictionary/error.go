package dictionary

import (
	_ "embed"

	errorpackage "github.com/siti-nabila/error-package"
)

var (
	//go:embed err_list.yaml
	errorList []byte

	errorPack = mustLoadErrorDictionary(errorList)

	ErrInvalidBrowserOrigins        = errorPack.New("invalid_browser_origins")
	ErrInvalidCSRFSecret            = errorPack.New("invalid_csrf_secret")
	ErrSessionDependencyRequired    = errorPack.New("session_dependency_required")
	ErrSessionRequestRequired       = errorPack.New("session_request_required")
	ErrSessionResultInvalid         = errorPack.New("session_result_invalid")
	ErrBrowserPolicyRequired        = errorPack.New("browser_policy_required")
	ErrCSRFGenerationFailed         = errorPack.New("csrf_generation_failed")
	ErrValueRequired                = errorPack.New("value_required")
	ErrPortOutOfRange               = errorPack.New("port_out_of_range")
	ErrValueNotPositive             = errorPack.New("value_not_positive")
	ErrAbsoluteURLRequired          = errorPack.New("absolute_url_required")
	ErrUnsupportedURLScheme         = errorPack.New("unsupported_url_scheme")
	ErrHTTPSRequired                = errorPack.New("https_required")
	ErrTLSEnabledForHTTPS           = errorPack.New("tls_enabled_for_https")
	ErrInvalidServiceToken          = errorPack.New("invalid_service_token")
	ErrServiceCredentialsRequireTLS = errorPack.New("service_credentials_require_tls")

	ErrApplicationConfigurationRequired   = errorPack.New("application_configuration_required")
	ErrListenContextRequired              = errorPack.New("listen_context_required")
	ErrHTTPRoundTripperRequired           = errorPack.New("http_round_tripper_required")
	ErrHTTPRequestContextRequired         = errorPack.New("http_request_context_required")
	ErrHTTPRequestRequired                = errorPack.New("http_request_required")
	ErrHTTPRequestURLRequired             = errorPack.New("http_request_url_required")
	ErrHTTPResponseBodyRequired           = errorPack.New("http_response_body_required")
	ErrGRPCProfileServiceRequired         = errorPack.New("grpc_profile_service_required")
	ErrGRPCProfileRequestRequired         = errorPack.New("grpc_profile_request_required")
	ErrGRPCProfileResponseRequired        = errorPack.New("grpc_profile_response_required")
	ErrGRPCPasswordServiceRequired        = errorPack.New("grpc_password_service_required")
	ErrGRPCChangePasswordRequestRequired  = errorPack.New("grpc_change_password_request_required")
	ErrGRPCChangePasswordResponseRequired = errorPack.New("grpc_change_password_response_required")
	ErrGRPCAuthTransportRequired          = errorPack.New("grpc_auth_transport_required")
	ErrGRPCAuthServiceRequired            = errorPack.New("grpc_auth_service_required")
	ErrGRPCAuthRequestContextRequired     = errorPack.New("grpc_auth_request_context_required")
	ErrGRPCAuthListUsersRequestRequired   = errorPack.New("grpc_auth_list_users_request_required")
	ErrGRPCAuthMeResponseRequired         = errorPack.New("grpc_auth_me_response_required")
	ErrGRPCAuthListUsersResponseRequired  = errorPack.New(
		"grpc_auth_list_users_response_required",
	)
	ErrUsersRepositoryClientRequired    = errorPack.New("users_repository_client_required")
	ErrUsersRepositoryContextRequired   = errorPack.New("users_repository_context_required")
	ErrUsersRepositoryListItemRequired  = errorPack.New("users_repository_list_item_required")
	ErrUsersGatewayClientRequired       = ErrUsersRepositoryClientRequired
	ErrUsersGatewayContextRequired      = ErrUsersRepositoryContextRequired
	ErrUsersGatewayListItemRequired     = ErrUsersRepositoryListItemRequired
	ErrAuthenticatorClientRequired      = errorPack.New("authenticator_client_required")
	ErrAuthenticatorContextRequired     = errorPack.New("authenticator_context_required")
	ErrAuthenticationMiddlewareRequired = errorPack.New(
		"authentication_middleware_required",
	)
	ErrUsersUseCaseRepositoryRequired     = errorPack.New("users_usecase_repository_required")
	ErrUsersUseCaseGatewayRequired        = ErrUsersUseCaseRepositoryRequired
	ErrUsersUseCaseContextRequired        = errorPack.New("users_usecase_context_required")
	ErrUsersHandlerUseCaseRequired        = errorPack.New("users_handler_usecase_required")
	ErrUsersHandlerResponseWriterRequired = errorPack.New(
		"users_handler_response_writer_required",
	)
	// Auth guards retain the existing error identities after the ownership move.
	ErrAuthGatewayClientRequired         = ErrUsersGatewayClientRequired
	ErrAuthGatewayContextRequired        = ErrUsersGatewayContextRequired
	ErrAuthUseCaseGatewayRequired        = ErrUsersUseCaseGatewayRequired
	ErrAuthUseCaseContextRequired        = ErrUsersUseCaseContextRequired
	ErrAuthHandlerUseCaseRequired        = ErrUsersHandlerUseCaseRequired
	ErrAuthHandlerResponseWriterRequired = errorPack.New("auth_handler_response_writer_required")
	ErrGRPCAuthRegisterRequestRequired   = errorPack.New("grpc_auth_register_request_required")
	ErrGRPCAuthRegisterResponseRequired  = errorPack.New("grpc_auth_register_response_required")
	ErrGRPCAuthLoginRequestRequired      = errorPack.New("grpc_auth_login_request_required")
	ErrGRPCAuthLoginResponseRequired     = errorPack.New("grpc_auth_login_response_required")
)

func mustLoadErrorDictionary(data []byte) errorpackage.DictionaryPack {
	errorpackage.SetLanguage(string(errorpackage.English))

	dictionary := errorpackage.NewErrYamlPackage()
	if err := dictionary.LoadBytes(data); err != nil {
		panic(err)
	}
	return dictionary
}
