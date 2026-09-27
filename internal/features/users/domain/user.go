package domain

type ListItem struct {
	ID      int32  `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Phone   string `json:"phone"`
}
