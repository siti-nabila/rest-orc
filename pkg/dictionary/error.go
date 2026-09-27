package dictionary

import (
	_ "embed"

	errorpackage "github.com/siti-nabila/error-package"
)

var (
	//go:embed err_list.yaml
	errorList []byte

	errorPack = mustLoadErrorDictionary(errorList)

	ErrValueRequired        = errorPack.New("value_required")
	ErrPortOutOfRange       = errorPack.New("port_out_of_range")
	ErrValueNotPositive     = errorPack.New("value_not_positive")
	ErrAbsoluteURLRequired  = errorPack.New("absolute_url_required")
	ErrUnsupportedURLScheme = errorPack.New("unsupported_url_scheme")
	ErrHTTPSRequired        = errorPack.New("https_required")
	ErrTLSEnabledForHTTPS   = errorPack.New("tls_enabled_for_https")

	ErrApplicationConfigurationRequired  = errorPack.New("application_configuration_required")
	ErrListenContextRequired             = errorPack.New("listen_context_required")
	ErrHTTPRoundTripperRequired          = errorPack.New("http_round_tripper_required")
	ErrHTTPRequestContextRequired        = errorPack.New("http_request_context_required")
	ErrHTTPRequestRequired               = errorPack.New("http_request_required")
	ErrHTTPRequestURLRequired            = errorPack.New("http_request_url_required")
	ErrHTTPResponseBodyRequired          = errorPack.New("http_response_body_required")
	ErrGRPCAuthTransportRequired         = errorPack.New("grpc_auth_transport_required")
	ErrGRPCAuthServiceRequired           = errorPack.New("grpc_auth_service_required")
	ErrGRPCAuthRequestContextRequired    = errorPack.New("grpc_auth_request_context_required")
	ErrGRPCAuthListUsersRequestRequired  = errorPack.New("grpc_auth_list_users_request_required")
	ErrGRPCAuthMeResponseRequired        = errorPack.New("grpc_auth_me_response_required")
	ErrGRPCAuthListUsersResponseRequired = errorPack.New(
		"grpc_auth_list_users_response_required",
	)
	ErrUsersRepositoryClientRequired    = errorPack.New("users_repository_client_required")
	ErrUsersRepositoryContextRequired   = errorPack.New("users_repository_context_required")
	ErrUsersRepositoryListItemRequired  = errorPack.New("users_repository_list_item_required")
	ErrAuthenticatorClientRequired      = errorPack.New("authenticator_client_required")
	ErrAuthenticatorContextRequired     = errorPack.New("authenticator_context_required")
	ErrAuthenticationMiddlewareRequired = errorPack.New(
		"authentication_middleware_required",
	)
	ErrUsersUseCaseRepositoryRequired     = errorPack.New("users_usecase_repository_required")
	ErrUsersUseCaseContextRequired        = errorPack.New("users_usecase_context_required")
	ErrUsersHandlerUseCaseRequired        = errorPack.New("users_handler_usecase_required")
	ErrUsersHandlerResponseWriterRequired = errorPack.New(
		"users_handler_response_writer_required",
	)
	ErrGRPCAuthRegisterRequestRequired  = errorPack.New("grpc_auth_register_request_required")
	ErrGRPCAuthRegisterResponseRequired = errorPack.New("grpc_auth_register_response_required")
	ErrGRPCAuthLoginRequestRequired     = errorPack.New("grpc_auth_login_request_required")
	ErrGRPCAuthLoginResponseRequired    = errorPack.New("grpc_auth_login_response_required")
)

func mustLoadErrorDictionary(data []byte) errorpackage.DictionaryPack {
	errorpackage.SetLanguage(string(errorpackage.English))

	dictionary := errorpackage.NewErrYamlPackage()
	if err := dictionary.LoadBytes(data); err != nil {
		panic(err)
	}
	return dictionary
}
