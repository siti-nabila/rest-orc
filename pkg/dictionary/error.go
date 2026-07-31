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

	ErrApplicationConfigurationRequired = errorPack.New("application_configuration_required")
	ErrListenContextRequired            = errorPack.New("listen_context_required")
	ErrHTTPRoundTripperRequired         = errorPack.New("http_round_tripper_required")
	ErrHTTPRequestContextRequired       = errorPack.New("http_request_context_required")
	ErrHTTPRequestRequired              = errorPack.New("http_request_required")
	ErrHTTPRequestURLRequired           = errorPack.New("http_request_url_required")
	ErrHTTPResponseBodyRequired         = errorPack.New("http_response_body_required")
)

func mustLoadErrorDictionary(data []byte) errorpackage.DictionaryPack {
	errorpackage.SetLanguage(string(errorpackage.English))

	dictionary := errorpackage.NewErrYamlPackage()
	if err := dictionary.LoadBytes(data); err != nil {
		panic(err)
	}
	return dictionary
}
