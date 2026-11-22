package update

import "testing"

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"v0.2.0", "0.1.0", true},
		{"v0.1.0", "0.1.0", false},
		{"v0.10.0", "0.9.0", true},
		{"v0.1.0", "0.1.0-rc.1", true},
		{"v0.1.0-rc.2", "0.1.0", false},
		{"v0.1.0-rc.10", "0.1.0-rc.2", true},
		{"v0.1.0-rc.2", "0.1.0-rc.10", false},
		{"v0.1.0-rc.1.1", "0.1.0-rc.1", true},
		{"garbage", "0.1.0", false},
		{"v9.0.0-x/../../tmp/evil", "0.1.0", false},
		{"v9.0.0-", "0.1.0", false},
		{"v9.0.0-rc..1", "0.1.0", false},
		{" v9.0.0\n", "0.1.0", false},
		{"v+9.0.0", "0.1.0", false},
		{"v9.0.+0", "0.1.0", false},
	} {
		if got := Newer(c.a, c.b); got != c.want {
			t.Errorf("Newer(%q, %q) = %v", c.a, c.b, got)
		}
	}
}
