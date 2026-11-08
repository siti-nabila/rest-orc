package fixtures

import (
	applicationv1 "github.com/siti-nabila/api-contracts/pb/application/v1"
	paginatorv1 "github.com/siti-nabila/api-contracts/pb/paginator/v1"
)

func ApplicationItem() *applicationv1.ApplicationItem {
	return &applicationv1.ApplicationItem{Id: 5, Code: "clm", Name: " Legacy NAME\té ", Status: "DISABLED", IsSystem: false}
}
func ApplicationList() *applicationv1.ListApplicationsResponse {
	return &applicationv1.ListApplicationsResponse{Items: []*applicationv1.ApplicationItem{ApplicationItem()}, Meta: &paginatorv1.PageMeta{Page: 1, Limit: 20, Total: 1, TotalPages: 1}}
}

const ApplicationJSON = `{"id":5,"code":"clm","name":" Legacy NAME\té ","status":"DISABLED","is_system":false}`
