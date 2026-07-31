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
	BadRequestIndonesianBody = `{"code":"BR","errors":{"email":["harus diisi"]},"data":[]}`
	InternalErrorBody        = `{"code":"IS","errors":{"description":"Internal server error."},"data":[]}`
)
