package entity

type Contest struct{}
type Subcontest struct{}
type Match struct{}
type GroupPick struct{}
type KnockoutPick struct{}
type LeaderboardEntry struct{}
type Group struct{}

type User struct {
	ID       string
	Email    string
	Username string
}
