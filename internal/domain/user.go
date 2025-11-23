package domain

type User struct {
	ID       string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

func NewUser(id, username, teamName string, isActive bool) *User {
	return &User{
		ID:       id,
		Username: username,
		TeamName: teamName,
		IsActive: isActive,
	}
}

func (u *User) Validate() error {
	if u.ID == "" {
		return ErrInvalidInput
	}
	if u.Username == "" {
		return ErrInvalidInput
	}
	if u.TeamName == "" {
		return ErrInvalidInput
	}
	return nil
}
