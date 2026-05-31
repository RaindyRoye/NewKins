package bean

import "testing"

func TestIsMatch(t *testing.T) {
	tests := []struct {
		name string
		s    string
		p    string
		want bool
	}{
		{"exact match", "hello", "hello", true},
		{"no match", "hello", "world", false},
		{"wildcard all", "hello", "*", true},
		{"wildcard prefix", "hello", "*llo", true},
		{"wildcard suffix", "hello", "hel*", true},
		{"wildcard middle", "hello", "h*o", true},
		{"wildcard no match", "hello", "x*", false},
		{"empty pattern empty string", "", "", true},
		{"empty string wildcard", "", "*", true},
		{"empty string non-wildcard", "", "a", false},
		{"multiple wildcards", "hello world", "h*d*", true},
		{"multiple wildcards no match", "hello", "x*y*", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMatch(tt.s, tt.p)
			if got != tt.want {
				t.Errorf("isMatch(%q, %q) = %v, want %v", tt.s, tt.p, got, tt.want)
			}
		})
	}
}

func TestCondition_Match_NilCondition(t *testing.T) {
	var c *Condition
	if c.Match("anything") {
		t.Error("nil Condition.Match should return false")
	}
}

func TestCondition_Includes(t *testing.T) {
	tests := []struct {
		name    string
		include []string
		val     string
		want    bool
	}{
		{"exact match", []string{"main", "develop"}, "main", true},
		{"no match", []string{"main", "develop"}, "feature", false},
		{"wildcard match", []string{"feature/*"}, "feature/login", true},
		{"wildcard no match", []string{"feature/*"}, "bugfix/login", false},
		{"empty pattern skipped", []string{"", "main"}, "main", true},
		{"all empty patterns", []string{"", ""}, "main", false},
		{"regex match", []string{"^release-.*"}, "release-1.0", true},
		{"regex no match", []string{"^release-.*"}, "feature-1.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Condition{Include: tt.include}
			got := c.Includes(tt.val)
			if got != tt.want {
				t.Errorf("Condition.Includes(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestCondition_Excludes(t *testing.T) {
	tests := []struct {
		name    string
		exclude []string
		val     string
		want    bool
	}{
		{"exact match", []string{"main"}, "main", true},
		{"no match", []string{"main"}, "develop", false},
		{"wildcard match", []string{"hotfix/*"}, "hotfix/123", true},
		{"wildcard no match", []string{"hotfix/*"}, "feature/123", false},
		{"empty pattern skipped", []string{""}, "main", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Condition{Exclude: tt.exclude}
			got := c.Excludes(tt.val)
			if got != tt.want {
				t.Errorf("Condition.Excludes(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestCondition_Match(t *testing.T) {
	tests := []struct {
		name    string
		cond    *Condition
		val     string
		want    bool
	}{
		{
			name: "include only, match",
			cond: &Condition{Include: []string{"main", "develop"}},
			val:  "main",
			want: true,
		},
		{
			name: "include only, no match",
			cond: &Condition{Include: []string{"main", "develop"}},
			val:  "feature",
			want: false,
		},
		{
			name: "exclude only, not excluded",
			cond: &Condition{Exclude: []string{"main"}},
			val:  "develop",
			want: true,
		},
		{
			name: "exclude only, excluded",
			cond: &Condition{Exclude: []string{"main"}},
			val:  "main",
			want: false,
		},
		{
			name: "both include and exclude, included and not excluded",
			cond: &Condition{
				Include: []string{"release/*"},
				Exclude: []string{"release/beta*"},
			},
			val:  "release/1.0",
			want: true,
		},
		{
			name: "both include and exclude, included but excluded",
			cond: &Condition{
				Include: []string{"release/*"},
				Exclude: []string{"release/beta*"},
			},
			val:  "release/beta1",
			want: false,
		},
		{
			name: "both include and exclude, not included",
			cond: &Condition{
				Include: []string{"release/*"},
				Exclude: []string{"release/beta*"},
			},
			val:  "feature/1.0",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cond.Match(tt.val)
			if got != tt.want {
				t.Errorf("Condition.Match(%q) = %v, want %v", tt.val, got, tt.want)
			}
		})
	}
}

func TestNewPipeline_Check(t *testing.T) {
	tests := []struct {
		name string
		p    *NewPipeline
		want bool
	}{
		{
			name: "valid pipeline",
			p:    &NewPipeline{Name: "test", Content: "stages: []"},
			want: true,
		},
		{
			name: "empty name",
			p:    &NewPipeline{Name: "", Content: "stages: []"},
			want: false,
		},
		{
			name: "empty content",
			p:    &NewPipeline{Name: "test", Content: ""},
			want: false,
		},
		{
			name: "both empty",
			p:    &NewPipeline{},
			want: false,
		},
		{
			name: "name only",
			p:    &NewPipeline{Name: "test"},
			want: false,
		},
		{
			name: "content only",
			p:    &NewPipeline{Content: "stages: []"},
			want: false,
		},
		{
			name: "all fields set",
			p: &NewPipeline{
				Name:        "test",
				Content:     "stages: []",
				DisplayName: "Test Pipeline",
				OrgId:       "org1",
				Url:         "https://github.com/test/repo",
				Username:    "user",
				AccessToken: "token",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.p.Check()
			if got != tt.want {
				t.Errorf("NewPipeline.Check() = %v, want %v", got, tt.want)
			}
		})
	}
}
