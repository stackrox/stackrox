package graph

var networkPolicyFixtureYAMLs = []string{
	`
kind: NetworkPolicy
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

`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: default-deny-all
  namespace: default
spec:
  podSelector: {}
  ingress: []
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: default-deny-all-egress
  namespace: default
spec:
  policyTypes:
  - Egress
  podSelector: {}
  egress: []
`,

	`
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: web-deny-external-egress
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  policyTypes:
  - Egress
  egress:
  - ports:
    - port: 53
      protocol: UDP
    - port: 53
      protocol: TCP
  - to:
    - namespaceSelector: {}
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  namespace: default
  name: deny-from-other-namespaces
spec:
  podSelector:
    matchLabels:
  ingress:
  - from:
    - podSelector: {}
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: limit-traffic
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: bookstore
      role: api
  ingress:
  - from:
      - podSelector:
          matchLabels:
            app: bookstore
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  namespace: default
  name: web-allow-all-namespaces
spec:
  podSelector:
    matchLabels:
      app: web
  ingress:
  - from:
    - namespaceSelector: {}
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: web-allow-all
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  ingress:
  - {}
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: web-allow-stackrox
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: stackrox
`,

	`
kind: NetworkPolicy
apiVersion: networking.k8s.io/v1
metadata:
  name: web-deny-all
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  ingress: []
`,

	`
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: web-deny-egress
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: web
  policyTypes:
  - Egress
  egress: []
`,
	`
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all-ingress
  namespace: qa
spec:
  podSelector: {}
  policyTypes:
  - Ingress
`,
	`
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-ingress-to-web
  namespace: qa
spec:
  ingress:
  - from:
    - namespaceSelector: {}
  podSelector:
    matchLabels:
      app: web

`,
}
