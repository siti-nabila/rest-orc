package domain

type MeResponse struct {
	ID                 uint64                `json:"id"`
	Email              string                `json:"email"`
	Fullname           string                `json:"fullname"`
	Address            string                `json:"address"`
	Phone              string                `json:"phone"`
	CountryCode        string                `json:"country_code"`
	CreatedAt          *string               `json:"created_at"`
	Status             string                `json:"status"`
	Version            string                `json:"version"`
	MustChangePassword bool                  `json:"must_change_password"`
	RoleCodes          []int32               `json:"role_codes"`
	RoleNames          []string              `json:"role_names"`
	Roles              []MeRole              `json:"roles"`
	ApplicationAccess  []MeApplicationAccess `json:"application_access"`
	Permissions        map[string]bool       `json:"permissions"`
}

type MeRole struct {
	RoleID     uint64 `json:"role_id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	RoleStatus string `json:"role_status"`
}

type MeApplicationAccess struct {
	ApplicationID     uint64 `json:"application_id"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	ApplicationStatus string `json:"application_status"`
}
