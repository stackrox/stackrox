package declarativeconfig

// KeyValuePair represents pair of key and value.
type KeyValuePair struct {
	Key   string `yaml:"key,omitempty"`
	Value string `yaml:"value,omitempty"`
}

// GenericConfig is representation of storage.Notifier_Generic that supports transformation from YAML.
type GenericConfig struct {
	Endpoint            string         `yaml:"endpoint,omitempty"`
	SkipTLSVerify       bool           `yaml:"skipTLSVerify,omitempty"`
	CACertPEM           string         `yaml:"caCertPEM,omitempty"`
	Username            string         `yaml:"username,omitempty"`
	Password            string         `yaml:"password,omitempty"`
	Headers             []KeyValuePair `yaml:"headers,omitempty"`
	ExtraFields         []KeyValuePair `yaml:"extraFields,omitempty"`
	AuditLoggingEnabled bool           `yaml:"auditLoggingEnabled,omitempty"`
}

// SourceTypePair represents a pair of key and which source type will be used for that key.
type SourceTypePair struct {
	Key   string `yaml:"key,omitempty"`
	Value string `yaml:"sourceType,omitempty"`
}

// SplunkConfig is representation of storage.Notifier_Splunk that supports transformation from YAML.
type SplunkConfig struct {
	HTTPToken           string           `yaml:"token,omitempty"`
	HTTPEndpoint        string           `yaml:"endpoint,omitempty"`
	Insecure            bool             `yaml:"skipTLSVerify,omitempty"`
	AuditLoggingEnabled bool             `yaml:"auditLoggingEnabled,omitempty"`
	Truncate            int64            `yaml:"hecTruncateLimit,omitempty"`
	SourceTypes         []SourceTypePair `yaml:"sourceTypes,omitempty"`
}

// Notifier is representation of storage.Notifier that supports transformation from YAML.
type Notifier struct {
	Name          string         `yaml:"name,omitempty"`
	GenericConfig *GenericConfig `yaml:"generic,omitempty"`
	SplunkConfig  *SplunkConfig  `yaml:"splunk,omitempty"`
}

// ConfigurationType returns the NotifierConfiguration type.
func (r *Notifier) ConfigurationType() ConfigurationType {
	return NotifierConfiguration
}
