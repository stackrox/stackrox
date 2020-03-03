package endpoints

import "strings"

// ParseLegacySpec parses a legacy specification string that is used for environment variables, i.e., a comma-separated
// list of `[type@]endpoint` strings (where type is `grpc` or `http`, and endpoint can be `<port>` or
// `[address]:<port>`).
func ParseLegacySpec(fullSpecStr string, tlsConfig *TLSConfig) []EndpointConfig {
	specs := strings.Split(fullSpecStr, ",")

	result := make([]EndpointConfig, 0, len(specs))
	for _, spec := range specs {
		spec = strings.TrimSpace(spec)
		if spec == "" {
			continue
		}

		var protos []string
		parts := strings.SplitN(spec, "@", 2)
		if len(parts) == 2 {
			protos = []string{strings.TrimSpace(parts[0])}
		}
		result = append(result, EndpointConfig{
			Listen:    strings.TrimSpace(parts[len(parts)-1]),
			Protocols: protos,
			TLS:       tlsConfig,
		})
	}
	return result
}
