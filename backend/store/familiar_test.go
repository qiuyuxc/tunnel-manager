package store

import (
	"fmt"
	"testing"
	"time"

	"tunnel-manager/db"
	"tunnel-manager/models"
)

func TestSubnetOfKeepsOnlyTheNeighbourhood(t *testing.T) {
	cases := []struct{ ip, want string }{
		{"203.0.113.57", "203.0.113.0/24"},
		{"203.0.113.9", "203.0.113.0/24"},
		{"203.0.114.9", "203.0.114.0/24"},
		{"2001:db8:1234:5678:9abc:def0:1234:5678", "2001:db8:1234:5678::/64"},
		{"::1", "::/64"},
		{"", ""},
		{"not-an-address", ""},
	}
	for _, tc := range cases {
		if got := SubnetOf(tc.ip); got != tc.want {
			t.Errorf("SubnetOf(%q) = %q, want %q", tc.ip, got, tc.want)
		}
	}
}

// The point of learning a /24 rather than the exact address: a dynamic address
// that moves inside the same pool still counts as the same place.
func TestFamiliarIPMatchesTheWholeSubnet(t *testing.T) {
	st := testStore(t)
	user := st.AdminUserID()

	if err := st.RememberFamiliarIP(user, "203.0.113.57"); err != nil {
		t.Fatalf("RememberFamiliarIP() error = %v", err)
	}
	if !st.IsFamiliarIP(user, "203.0.113.200") {
		t.Fatal("another address in the same /24 was not familiar")
	}
	if st.IsFamiliarIP(user, "203.0.114.57") {
		t.Fatal("an address from a different /24 was familiar")
	}
	if st.IsFamiliarIP(user, "2001:db8::1") {
		t.Fatal("an IPv6 address matched an IPv4 network")
	}
}

func TestFamiliarIPIsPerAccount(t *testing.T) {
	st := testStore(t)
	if err := st.RememberFamiliarIP(st.AdminUserID(), "203.0.113.57"); err != nil {
		t.Fatal(err)
	}
	if st.IsFamiliarIP("someone-else", "203.0.113.57") {
		t.Fatal("one account's network was familiar to another")
	}
	if st.IsFamiliarIP("", "203.0.113.57") {
		t.Fatal("an empty account id was treated as familiar")
	}
	if st.IsFamiliarIP(st.AdminUserID(), "") {
		t.Fatal("an empty address was treated as familiar")
	}
}

func TestRememberFamiliarIPIgnoresUnparseableAddresses(t *testing.T) {
	st := testStore(t)
	if err := st.RememberFamiliarIP(st.AdminUserID(), "garbage"); err != nil {
		t.Fatalf("RememberFamiliarIP() error = %v", err)
	}
	networks, err := st.ListFamiliarIPs(st.AdminUserID())
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 0 {
		t.Fatalf("stored %v, want nothing for an unparseable address", networks)
	}
}

// Repeated sign-ins from one place must not pile up rows, and a roaming user
// must not grow the table without bound.
func TestRememberFamiliarIPPrunesAndDedupes(t *testing.T) {
	st := testStore(t)
	user := st.AdminUserID()

	for i := 0; i < 5; i++ {
		if err := st.RememberFamiliarIP(user, "203.0.113.57"); err != nil {
			t.Fatal(err)
		}
	}
	networks, err := st.ListFamiliarIPs(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 1 {
		t.Fatalf("stored %v, want the one network", networks)
	}

	for i := 0; i < maxFamiliarSubnetsPerUser+20; i++ {
		if err := st.RememberFamiliarIP(user, fmt.Sprintf("198.51.%d.7", i)); err != nil {
			t.Fatal(err)
		}
	}
	networks, err = st.ListFamiliarIPs(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) > maxFamiliarSubnetsPerUser {
		t.Fatalf("kept %d networks, want at most %d", len(networks), maxFamiliarSubnetsPerUser)
	}
}

func TestDeleteUserForgetsLearnedNetworks(t *testing.T) {
	st := testStore(t)
	if err := st.CreateUser(models.User{Username: "bob", PasswordHash: HashPassword("password")}); err != nil {
		t.Fatal(err)
	}
	var id string
	for _, u := range st.ListUsers() {
		if u.Username == "bob" {
			id = u.ID
		}
	}
	if id == "" {
		t.Fatal("created account not found")
	}
	if err := st.RememberFamiliarIP(id, "203.0.113.57"); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteUser(id); err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if st.IsFamiliarIP(id, "203.0.113.57") {
		t.Fatal("learned networks survived the account")
	}
}

// Trust has to expire: a network visited once must not widen the budget for
// whoever uses it next, months later.
func TestFamiliarIPExpiresAfterTheWindow(t *testing.T) {
	st := testStore(t)
	user := st.AdminUserID()
	if err := st.RememberFamiliarIP(user, "203.0.113.57"); err != nil {
		t.Fatal(err)
	}
	if !st.IsFamiliarIP(user, "203.0.113.57") {
		t.Fatal("a network just learned was not familiar")
	}

	handle, err := db.Open(st.dsn)
	if err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-familiarWindow - time.Hour).Unix()
	if _, err := handle.Exec(`UPDATE familiar_ips SET last_seen = ? WHERE user_id = ?`, stale, user); err != nil {
		t.Fatal(err)
	}
	handle.Close()

	if st.IsFamiliarIP(user, "203.0.113.57") {
		t.Fatal("a network last seen outside the window was still familiar")
	}

	// Signing in again refreshes it, which is what makes the window self-healing
	// for a user whose address moved.
	if err := st.RememberFamiliarIP(user, "203.0.113.57"); err != nil {
		t.Fatal(err)
	}
	if !st.IsFamiliarIP(user, "203.0.113.57") {
		t.Fatal("a refreshed network was not familiar")
	}
	networks, err := st.ListFamiliarIPs(user)
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 1 {
		t.Fatalf("stored %v, want the one refreshed network", networks)
	}
}
