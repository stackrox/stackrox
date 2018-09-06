package fixtures

// GetYAML returns a network policy yaml
func GetYAML() string {

	return ` kind: NetworkPolicy
			apiVersion: networking.k8s.io/v1
			metadata:
				name: allow-traffic-from-apps-using-multiple-selectors
			spec:
				podSelector:
					matchLabels:
						app: web
						role: db
				ingress:
					- from:
						- podSelector:
							matchLabels:
								app: bookstore
								role: search
						- podSelector:
							matchLabels:
								app: bookstore
								role: api
	`

}
