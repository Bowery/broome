package requests

import (
	"github.com/Bowery/gopackages/schemas"
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
	Developer *schemas.Developer `json:"developer"`
}

// copied from bowery/requests/bodies
type LoginReq struct {
	Name     string `json:"name,omitempty"` // Only some use name.
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PaymentReq struct {
	StripeToken string `json:"stripeToken"`
}
