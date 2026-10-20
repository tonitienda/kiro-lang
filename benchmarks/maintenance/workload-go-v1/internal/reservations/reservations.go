package reservations

type Status struct {
	Active bool `json:"active"`
}

func Expired(now, expiresAt int32) bool    { return now > expiresAt }
func StatusAt(now, expiresAt int32) Status { return Status{Active: !Expired(now, expiresAt)} }
