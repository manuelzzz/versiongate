package version

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major, Minor, Patch int
}

var ErrInvalidVersion = errors.New("invalid version")

func Parse(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version: %s", s)
	}

	nums := make([]int, 3)
	for i, p := range parts {
		n, err := parseNonNegativeInt(p)
		if err != nil {
			return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
		}
		nums[i] = n
	}

	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}, nil
}

func ParseBuildNumber(s string) (int, error) {
	n, err := parseNonNegativeInt(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidVersion, s)
	}
	return n, nil
}

func parseNonNegativeInt(s string) (int, error) {
	if s == "" {
		return 0, ErrInvalidVersion
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, ErrInvalidVersion
		}
	}
	n, err := strconv.Atoi(s)

	if err != nil {
		return 0, ErrInvalidVersion
	}

	return n, nil
}

func CompareWithBuild(v1 Version, build1 int, v2 Version, build2 int) int {
	if c := v1.Compare(v2); c != 0 {
		return c
	}
	return compareInt(build1, build2)
}

func (v Version) Compare(other Version) int {
	if c := compareInt(v.Major, other.Major); c != 0 {
		return c
	}
	if c := compareInt(v.Minor, other.Minor); c != 0 {
		return c
	}
	return compareInt(v.Patch, other.Patch)
}

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
