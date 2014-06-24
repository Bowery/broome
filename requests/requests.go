package requests

import (
	"github.com/Bowery/broome/db"
)

type Res struct {
	Status string `json:"status"`
	Err    string `json:"error"`
}

func (res *Res) Error() string {
	return res.Err
}

type DeveloperRes struct {
	*Res
	Developer *db.Developer `json:"developer"`
}
