package update

import (
	"strconv"
	"strings"
)

// Newer reports whether release version a is newer than b. Both are
// vMAJOR.MINOR.PATCH with an optional -prerelease; a prerelease sorts before
// its release. Anything unparsable is never newer.
func Newer(a, b string) bool {
	pa, oka := parseVersion(a)
	pb, okb := parseVersion(b)
	if !oka || !okb {
		return false
	}
	for i := range 3 {
		if pa.n[i] != pb.n[i] {
			return pa.n[i] > pb.n[i]
		}
	}
	switch {
	case pa.pre == pb.pre:
		return false
	case pa.pre == "":
		return true
	case pb.pre == "":
		return false
	}
	return comparePre(pa.pre, pb.pre) > 0
}

// comparePre orders prerelease strings by dot-separated identifier,
// numerically where both are numbers (semver §11), so rc.10 follows rc.2.
func comparePre(a, b string) int {
	xs, ys := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(xs) && i < len(ys); i++ {
		x, errx := strconv.Atoi(xs[i])
		y, erry := strconv.Atoi(ys[i])
		switch {
		case errx == nil && erry == nil:
			if x != y {
				return x - y
			}
		case xs[i] != ys[i]:
			return strings.Compare(xs[i], ys[i])
		}
	}
	return len(xs) - len(ys)
}

type version struct {
	n   [3]int
	pre string
}

func parseVersion(s string) (version, bool) {
	s = strings.TrimPrefix(s, "v")
	var v version
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, v.pre = s[:i], s[i+1:]
		if !prerelease(v.pre) {
			return v, false
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return v, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || strings.Trim(p, "0123456789") != "" {
			return v, false
		}
		v.n[i] = n
	}
	return v, true
}

// prerelease reports whether pre is a semver prerelease: dot-separated
// [0-9A-Za-z-] identifiers. A tag ends up in a URL and a file name, so
// parseVersion accepts only digits, dots and these.
func prerelease(pre string) bool {
	for _, id := range strings.Split(pre, ".") {
		if id == "" || strings.Trim(id, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-") != "" {
			return false
		}
	}
	return true
}
