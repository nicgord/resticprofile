package config

import (
	"reflect"
	"regexp"

	"github.com/creativeprojects/resticprofile/shell"
)

const ConfidentialReplacement = "×××"

// ConfidentialValue is a string value with a public and a confidential representation
type ConfidentialValue struct {
	public, confidential string
}

func (c ConfidentialValue) Value() string {
	return c.confidential
}

func (c ConfidentialValue) String() string {
	return c.public
}

func (c *ConfidentialValue) IsConfidential() bool {
	return c.public != c.confidential
}

// hideValue hides the entire value in the public representation
func (c *ConfidentialValue) hideValue() {
	if c.IsConfidential() {
		return
	}
	c.public = ConfidentialReplacement
}

// hideSubmatches hides all occurrences of submatches in the public representation
func (c *ConfidentialValue) hideSubmatches(pattern *regexp.Regexp) {
	if c.IsConfidential() {
		return
	}

	if matches := pattern.FindStringSubmatchIndex(c.confidential); len(matches) > 2 {
		c.public = c.confidential

		for i := len(matches) - 2; i > 1; i -= 2 {
			start := matches[i]
			end := matches[i+1]

			c.public = c.public[0:start] + ConfidentialReplacement + c.public[end:]
		}
	}
}

func NewConfidentialValue(value string) ConfidentialValue {
	return ConfidentialValue{public: value, confidential: value}
}

// confidentialValueDecoder implements parsing config parsing for ConfidentialValue
func confidentialValueDecoder() func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
	confidentialValueType := reflect.TypeOf(ConfidentialValue{})

	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if to != confidentialValueType {
			return data, nil
		}

		// Source type may be boolean, numeric or string
		values, isset := stringifyValue(reflect.ValueOf(data))
		if len(values) == 0 {
			if isset {
				values = []string{"1"}
			} else {
				values = []string{"0"}
			}
		}

		return NewConfidentialValue(values[0]), nil
	}
}

// See https://restic.readthedocs.io/en/latest/030_preparing_a_new_repo.html
var (
	urlConfidentialPart = regexp.MustCompile("[:/][^:/@]+?:([^:@]+?)@[^:/@]+?") // user:pass@host
	urlEnvKeys          = regexp.MustCompile("(?i)^.+(_AUTH|_URL)$")
	hiddenEnvKeys       = regexp.MustCompile("(?i)^(.+_KEY|.+_TOKEN|.*PASSWORD.*|.*SECRET.*)$")
)

// ProcessConfidentialValues hides confidential parts inside the specified Profile.
func ProcessConfidentialValues(profile *Profile) {
	if profile == nil {
		return
	}

	// Handle the repo URL
	profile.Repository.hideSubmatches(urlConfidentialPart)

	// Handle env variables
	for name, value := range profile.Environment {
		if hiddenEnvKeys.MatchString(name) {
			value.hideValue()
			profile.Environment[name] = value
		} else if urlEnvKeys.MatchString(name) {
			value.hideSubmatches(urlConfidentialPart)
			profile.Environment[name] = value
		}
	}
}

func getAllConfidentialValues(profile *Profile) []*ConfidentialValue {
	var confidentials []*ConfidentialValue

	if profile != nil {
		confidentials = append(confidentials, &profile.Repository)
		for _, value := range profile.Environment {
			confidentials = append(confidentials, &value)
		}
	}

	return confidentials
}

func convertToNonConfidential(confidentials []*ConfidentialValue, value string) string {
	for _, c := range confidentials {
		if c != nil && c.IsConfidential() && value == c.Value() {
			return c.String()
		}
	}
	return value
}

// GetNonConfidentialValues returns a new list with confidential values being replaced with their public representation
func GetNonConfidentialValues(profile *Profile, values []string) []string {
	target := make([]string, len(values))
	confidentials := getAllConfidentialValues(profile)

	for i := len(values) - 1; i >= 0; i-- {
		target[i] = convertToNonConfidential(confidentials, values[i])
	}

	return target
}

// GetNonConfidentialArgs returns new shell.Args with confidential values being replaced with their public representation
func GetNonConfidentialArgs(profile *Profile, args *shell.Args) *shell.Args {
	if args == nil {
		return nil
	}

	target := args.Clone()
	confidentials := getAllConfidentialValues(profile)

	target.Walk(func(name string, arg *shell.Arg) *shell.Arg {
		if arg.HasValue() {
			value := convertToNonConfidential(confidentials, arg.Value())
			if value != arg.Value() {
				a := shell.NewArg(value, arg.Type())
				return &a
			}
		}
		return arg
	})

	return target
}
