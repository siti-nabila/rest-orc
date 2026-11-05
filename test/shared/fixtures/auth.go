package fixtures

const (
	RegisterSuccessBody = `{"code":"SS","message":"Register Successfully","data":{"token":"","registration_id":17,"registration_status":"PENDING"}}`
	LoginSuccessBody    = `{"code":"SS","message":"Login Successfully","data":{"token":"login-token","token_type":"Bearer","session_id":"00000000-0000-4000-8000-000000000017","access_expires_at":"2030-01-01T00:15:00Z","session_expires_at":"2030-01-02T00:00:00Z","must_change_password":false}}`
)
