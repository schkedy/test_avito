package domain

type Team struct {
	Name    string `json:"team_name"`
	Members []User `json:"members"`
}

func NewTeam(name string, members []User) *Team {
	return &Team{
		Name:    name,
		Members: members,
	}
}

func (t *Team) Validate() error {
	if t.Name == "" {
		return ErrInvalidInput
	}
	return nil
}
