package config

// Data structures for the config feature

// ConfigVar represents a variable definition from the JSON config file
type ConfigVar struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Value       *string `json:"value"`
}

// ConfigSection represents a section from the JSON config file
type ConfigSection struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Variables   []ConfigVar `json:"variables"`
}

// ConfigRoot represents the root structure of the JSON config file
type ConfigRoot struct {
	Prefix   string          `json:"prefix"`
	Sections []ConfigSection `json:"sections"`
}

// EnvVar represents a single environment variable with its metadata
type EnvVar struct {
	Name        string
	Type        string
	Description string
	Value       *string
}

// EnvVarSection represents a logical grouping of environment variables
type EnvVarSection struct {
	Name        string
	Description string
	Vars        []EnvVar
}

// EnvVarRoot is the root configuration containing all sections
type EnvVarRoot struct {
	Prefix   string
	Sections []EnvVarSection
}
