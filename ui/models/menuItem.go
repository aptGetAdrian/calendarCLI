package models

//////////////////////////////////////
// Menu interface for all app menus
//////////////////////////////////////

type MenuItem struct {
	TitleValue string `json:"title"`
	Desc       string `json:"description"`
	Action     string `json:"action"`
}

func (i MenuItem) Title() string       { return i.TitleValue }
func (i MenuItem) Description() string { return i.Desc }
func (i MenuItem) FilterValue() string { return i.TitleValue }

func (i MenuItem) GetAction() string { return i.Action }
