package cmdspec

import "testing"

func TestSingular(t *testing.T) {
	for in, want := range map[string]string{"vms": "vm", "policies": "policy", "addresses": "address", "boxes": "box",
		"access": "access", "ssh-keys": "ssh-key", "nic": "nic", "ip-addresses": "ip-address"} {
		if got := Singular(in); got != want {
			t.Errorf("Singular(%q) = %q, want %q", in, got, want)
		}
	}
}
