package entity

import "time"

const (
	NumGroups = 12
	GroupSize = 4
)

type Contest struct {
	ID                 string
	Title              string
	Slug               string
	GroupUnlockDate    time.Time
	GroupLockDate      time.Time
	KnockoutUnlockDate time.Time
	KnockoutLockDate   time.Time
	Groups             []Group
}
type Subcontest struct{}
type Match struct {
	Country1          *Country
	Country2          *Country
	Country1Goals     *int
	Country2Goals     *int
	Country1Penalties *int
	Country2Penalties *int
	Round             int
	RoundIndex        *int
}
type GroupPick struct{}
type KnockoutPick struct{}
type LeaderboardEntry struct{}
type Group struct {
	Letter    string
	Countries []Country
}

type Country struct {
	Code     string
	FullName string
}
type User struct {
	ID       string
	Email    string
	Username string
}
