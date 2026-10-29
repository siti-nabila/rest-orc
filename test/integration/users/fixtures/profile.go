package fixtures

import profilev1 "github.com/siti-nabila/api-contracts/pb/profile/v1"

const ProfileSuccessBody = `{"code":"SS","message":"Profile updated successfully.","data":{"changed":true,"version":"9007199254740993","profile":{"id":9,"user_id":17,"name":"NASA O'Neil","address":"Jakarta","phone":"812345678901","country_code":"+62"}}}`
const ProfileNoOpBody = `{"code":"SS","message":"Profile updated successfully.","data":{"changed":false}}`

func ProfileResponse() *profilev1.ProfileResponse {
	return &profilev1.ProfileResponse{Changed: true, Version: 9007199254740993, Profile: &profilev1.Profile{Id: 9, UserId: 17, Name: "NASA O'Neil", Address: "Jakarta", Phone: "812345678901", CountryCode: "+62"}}
}
