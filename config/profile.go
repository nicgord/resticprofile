package config

import (
	"fmt"
	"path/filepath"

	"github.com/creativeprojects/resticprofile/constants"
	"github.com/spf13/viper"
)

type Profile struct {
	Name         string
	Quiet        bool                   `mapstructure:"quiet" argument:"quiet"`
	Verbose      bool                   `mapstructure:"verbose" argument:"verbose"`
	Repository   string                 `mapstructure:"repository" argument:"repo"`
	PasswordFile string                 `mapstructure:"password-file" argument:"password-file"`
	Initialize   bool                   `mapstructure:"initialize"`
	Inherit      string                 `mapstructure:"inherit"`
	Lock         bool                   `mapstructure:"lock"`
	Environment  map[string]string      `mapstructure:"env"`
	Backup       *BackupSection         `mapstructure:"backup"`
	Retention    *RetentionSection      `mapstructure:"retention"`
	Snapshots    map[string]interface{} `mapstructure:"snapshots"`
	Forget       map[string]interface{} `mapstructure:"forget"`
	Check        map[string]interface{} `mapstructure:"check"`
	Mount        map[string]interface{} `mapstructure:"mount"`
	OtherFlags   map[string]interface{} `mapstructure:",remain"`
}

type BackupSection struct {
	CheckBefore bool                   `mapstructure:"check-before"`
	CheckAfter  bool                   `mapstructure:"check-after"`
	RunBefore   []string               `mapstructure:"run-before"`
	RunAfter    []string               `mapstructure:"run-after"`
	UseStdin    bool                   `mapstructure:"stdin"`
	Source      []string               `mapstructure:"source"`
	OtherFlags  map[string]interface{} `mapstructure:",remain"`
}

type RetentionSection struct {
	BeforeBackup bool                   `mapstructure:"before-backup"`
	AfterBackup  bool                   `mapstructure:"after-backup"`
	OtherFlags   map[string]interface{} `mapstructure:",remain"`
}

func NewProfile(name string) *Profile {
	return &Profile{
		Name: name,
	}
}

func LoadProfile(profileKey string) (*Profile, error) {
	var err error
	var profile *Profile

	if !viper.IsSet(profileKey) {
		return nil, nil
	}

	// Load parent profile first if it is inherited
	if viper.IsSet(profileKey + "." + constants.ParameterInherit) {
		inherit := viper.GetString(profileKey + "." + constants.ParameterInherit)
		if inherit != "" {
			profile, err = LoadProfile(inherit)
			if err != nil {
				return nil, err
			}
			if profile == nil {
				return nil, fmt.Errorf("Error in profile '%s': Parent profile '%s' not found", profileKey, inherit)
			}
			profile.Name = profileKey
		}
	}

	// If profile is not inherited, create a blank one
	if profile == nil {
		profile = NewProfile(profileKey)
	}

	err = viper.UnmarshalKey(profileKey, profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (p *Profile) SetRootPath(rootPath string) {
	p.PasswordFile = filepath.Join(rootPath, p.PasswordFile)
}

func (p *Profile) GetCommonFlags() map[string][]string {
	// Flags from the profile fields
	flags := convertStructToFlags(*p)

	flags = addOtherFlags(flags, p.OtherFlags)

	return flags
}

func (p *Profile) GetCommandFlags(command string) map[string][]string {
	flags := p.GetCommonFlags()

	switch command {
	case constants.CommandBackup:
		commandFlags := convertStructToFlags(*p.Backup)
		if commandFlags != nil && len(commandFlags) > 0 {
			flags = mergeFlags(flags, commandFlags)
		}
		flags = addOtherFlags(flags, p.Backup.OtherFlags)
	}
	return flags
}

func (p *Profile) GetRetentionFlags() []string {
	flags := make([]string, 0)
	return flags
}

func (p *Profile) GetBackupSource() []string {
	return p.Backup.Source
}

func addOtherFlags(flags map[string][]string, otherFlags map[string]interface{}) map[string][]string {
	if otherFlags == nil || len(otherFlags) == 0 {
		return flags
	}

	// Add other flags
	for name, value := range otherFlags {
		if convert, ok := stringifyValueOf(value); ok {
			flags[name] = convert
		}
	}
	return flags
}

func mergeFlags(flags, newFlags map[string][]string) map[string][]string {
	if (flags == nil || len(flags) == 0) && newFlags != nil {
		return newFlags
	}
	if flags != nil && (newFlags == nil || len(newFlags) == 0) {
		return flags
	}
	for key, value := range newFlags {
		flags[key] = value
	}
	return flags
}
