package fixtures

import (
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
	sessionv1 "github.com/siti-nabila/api-contracts/pb/session/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

const MySessionID = "abcdefab-abcd-abcd-abcd-abcdefabcdef"

func MySessionsResponse(owner uint64) *sessionv1.ListSessionsResponse {
	return &sessionv1.ListSessionsResponse{Items: []*sessionv1.Session{{Id: MySessionID, UserId: owner, Status: "ACTIVE", CreatedAt: timestamppb.New(time.Date(2020, 1, 1, 0, 0, 0, 123456789, time.UTC)), ExpiresAt: timestamppb.New(time.Date(2020, 1, 8, 0, 0, 0, 0, time.UTC)), IsCurrent: false}}, Meta: &paginatorv1.PageMeta{Page: 1, Limit: 20, Total: 2147483648, TotalPages: 107374183, HasNext: true, HasPrev: true}, NextCursor: "opaque-next"}
}
