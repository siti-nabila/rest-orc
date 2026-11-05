package fixtures

type Profile struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

var UserProfile = Profile{
	ID:       1024,
	Username: "johndoe",
}

const (
	SuccessBody              = `{"code":"SS","message":"User profile retrieved successfully.","data":{"id":1024,"username":"johndoe"}}`
	NotFoundIndonesianBody   = `{"code":"NF","errors":{"description":"Data tidak ditemukan."},"data":[]}`
	EndpointNotFoundBody     = `{"code":"NF","errors":{"description":"Endpoint tidak ditemukan."},"data":[]}`
	BadRequestIndonesianBody = `{"code":"BR","errors":{"email":["harus diisi"]},"data":[]}`
	AlreadyExistsEnglishBody = `{"code":"EX","errors":{"description":"Already exists."},"data":[]}`
	AlreadyExistsChineseBody = `{"code":"EX","errors":{"description":"已存在。"},"data":[]}`
	BadRequestChineseBody    = `{"code":"BR","errors":{"email":["必填","最小长度为 6"]},"data":[]}`
	DeadlineExceededBody     = `{"code":"TO","errors":{"description":"Request timeout."},"data":[]}`
	InternalErrorBody        = `{"code":"IS","errors":{"description":"Internal server error."},"data":[]}`
)
