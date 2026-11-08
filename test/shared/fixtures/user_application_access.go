package fixtures

import (
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	userv1 "github.com/siti-nabila/api-contracts/pb/user/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func UserAccessHistory(target uint64) *userv1.ListUserApplicationAccessResponse {
	return &userv1.ListUserApplicationAccessResponse{UserId: target, Version: 9007199254740993, Items: []*userv1.UserApplicationAccessItem{{ApplicationId: 5, ApplicationCode: "clm", Name: " Legacy NAME\té ", ApplicationStatus: "DISABLED", Status: "REVOKED", GrantedAt: timestamppb.New(MeCreatedAt())}}, Meta: &paginatorv1.PageMeta{Page: 1, Limit: 20, Total: 1, TotalPages: 1}}
}

const UserAccessHistoryJSON = `{"application_id":5,"application_code":"clm","name":" Legacy NAME\té ","application_status":"DISABLED","is_system":false,"status":"REVOKED","granted_by":null,"granted_at":"2026-10-20T08:09:10.123456789Z","revoked_by":null,"revoked_at":null,"revoke_reason":null}`
