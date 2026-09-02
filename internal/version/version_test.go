package version

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr bool
	}{
		{name: "simple", input: "1.0.0", want: Version{1, 0, 0}},
		{name: "multi-digit minor", input: "1.10.0", want: Version{1, 10, 0}},
		{name: "empty", input: "", wantErr: true},
		{name: "two components", input: "1.0", wantErr: true},
		{name: "four components", input: "1.0.0.0", wantErr: true},
		{name: "non-numeric", input: "1.a.0", wantErr: true},
		{name: "negative", input: "1.-1.0", wantErr: true},
		{name: "v-prefixed", input: "v1.0.0", wantErr: true},
		{name: "leading whitespace", input: " 1.0.0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %v, want error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) returned unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("Parse(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b string // parsed via Parse in the test body
		want int
	}{
		{name: "numeric not lexicographic", a: "1.10.0", b: "1.2.0", want: 1},
		{name: "equal", a: "1.0.0", b: "1.0.0", want: 0},
		{name: "patch difference", a: "1.0.0", b: "1.0.1", want: -1},
		{name: "major dominates", a: "2.0.0", b: "1.10.10", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			va, err := Parse(tt.a)
			if err != nil {
				t.Fatalf("setup: Parse(%q) failed: %v", tt.a, err)
			}
			vb, err := Parse(tt.b)
			if err != nil {
				t.Fatalf("setup: Parse(%q) failed: %v", tt.b, err)
			}
			if got := va.Compare(vb); got != tt.want {
				t.Fatalf("Compare(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCompareWithBuild(t *testing.T) {
	v1 := Version{1, 0, 0}
	v2 := Version{1, 0, 1}

	tests := []struct {
		name   string
		v1     Version
		build1 int
		v2     Version
		build2 int
		want   int
	}{
		{name: "same version, higher build wins", v1: v1, build1: 2, v2: v1, build2: 1, want: 1},
		{name: "same version, same build", v1: v1, build1: 5, v2: v1, build2: 5, want: 0},
		{name: "higher build never beats higher version", v1: v1, build1: 99, v2: v2, build2: 1, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareWithBuild(tt.v1, tt.build1, tt.v2, tt.build2)
			if got != tt.want {
				t.Fatalf("CompareWithBuild(...) = %d, want %d", got, tt.want)
			}
		})
	}
}
