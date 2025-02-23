.ONESHELL:

NAMESPACE = stackrox
MAIN_IMAGE_TAG ?= null

.PHONY: clean
clean:
	rm -rf \
		ca-serial.txt ca-serial.txt* ca-db.txt* newcerts \
		ca-{cert,key}.pem crs-{key,csr,cert}.pem \
		cluster-registration-secret.{yaml,json} \
		stackrox-{central,secured-cluster}-services-chart \
		helm-values-*-services.yaml

.PHONY: helm-charts
helm-charts: stackrox-central-services-chart stackrox-secured-cluster-services-chart

stackrox-central-services-chart:
	roxctl helm output central-services --remove --debug

stackrox-secured-cluster-services-chart:
	roxctl helm output secured-cluster-services --remove --debug

define crs_json
{
  "version": 1,
  "CAs": ["$$CA_CERT"],
  "cert": "$$CRS_CERT",
  "key": "$$CRS_KEY"
}
endef
export crs_json

define crs_yaml
apiVersion: v1
kind: Secret
metadata:
  name: cluster-registration-secret
data:
  crs: $$OPAQUE_CRS
endef
export crs_yaml

.PHONY: ca
ca: ca-key.pem ca-cert.pem

ca-db.txt:
	echo -n > $@

ca-serial.txt:
	echo 01 > $@

newcerts:
	mkdir -p newcerts

ca-key.pem:
	openssl genpkey -algorithm RSA -out ca-key.pem

ca-cert.pem: ca-key.pem ca-db.txt ca-serial.txt newcerts
	openssl req -new -x509 -days 2 -key ca-key.pem -sha256 -out ca-cert.pem -subj "/CN=StackRox Certificate Authority"

crs-key.pem:
	openssl genpkey -algorithm RSA -out $@

crs-csr.pem: crs-key.pem
	export CRS_ID=$$(uuidgen | tr '[:upper:]' '[:lower:]')
	openssl req -new -key $< -out $@ -subj "/O=$$CRS_ID/OU=REGISTRANT_SERVICE/CN=REGISTRANT_SERVICE: 00000000-0000-0000-0000-000000000000"

crs-cert.pem: crs-csr.pem ca-key.pem
	openssl ca -config ./ca-config.txt -batch -notext -in $< -out $@

cluster-registration-secret.json: ca-cert.pem crs-key.pem crs-cert.pem
	@export CA_CERT="$$(echo "$$(cat ca-cert.pem)" | tr '\n' '|' | sed -e 's/|/\\\\n/g;')"
	@export CRS_CERT="$$(echo "$$(cat crs-cert.pem)" | tr '\n' '|' | sed -e 's/|/\\\\n/g;')"
	@export CRS_KEY="$$(echo "$$(cat crs-key.pem)" | tr '\n' '|' | sed -e 's/|/\\\\n/g;')"
	@echo "$${crs_json}"                       \
		| sed -e "s|\$$CA_CERT|$$CA_CERT|g;"   \
		| sed -e "s|\$$CRS_CERT|$$CRS_CERT|g;" \
		| sed -e "s|\$$CRS_KEY|$$CRS_KEY|g;"   \
		> $@

cluster-registration-secret.yaml: cluster-registration-secret.json
	@export OPAQUE_CRS=$$(cat cluster-registration-secret.json | base64 | tr -d '\n' | base64 | tr -d '\n')
	@echo "$${crs_yaml}"                           \
		| sed -e "s|\$$OPAQUE_CRS|$$OPAQUE_CRS|g;" \
		> $@

.PHONY: helm-values
helm-values: helm-values-central-services.yaml helm-values-secured-cluster-services.yaml

helm-values-central-services.yaml: base-helm-values-central-services.yaml ca-cert.pem ca-key.pem
	@export CA_CERT=$$(tr < ca-cert.pem '\n' '|' | sed -e 's/|/\\n/g;')
	export CA_KEY=$$(tr < ca-key.pem '\n' '|' | sed -e 's/|/\\n/g;')
	export MAIN_IMAGE_TAG=$(MAIN_IMAGE_TAG)
	echo "Generating $@..."
	envsubst < base-helm-values-central-services.yaml > helm-values-central-services.yaml \
		|| rm -f helm-values-central-services.yaml

helm-values-secured-cluster-services.yaml: base-helm-values-secured-cluster-services.yaml cluster-registration-secret.yaml
	@export CRS=$$(tr < cluster-registration-secret.yaml '\n' '|' | sed -e 's/|/\\n/g;')
	export MAIN_IMAGE_TAG=$(MAIN_IMAGE_TAG)
	envsubst < base-helm-values-secured-cluster-services.yaml > helm-values-secured-cluster-services.yaml \
		|| rm -f helm-values-secured-cluster-services.yaml

.PHONY: install-central-services
install-central-services: stackrox-central-services-chart helm-values-central-services.yaml
	helm install -n $(NAMESPACE) -f helm-values-central-services.yaml stackrox-central-services ./stackrox-central-services-chart

.PHONY: uninstall-central-services
uninstall-central-services:
	@helm uninstall -n $(NAMESPACE) stackrox-central-services || true
	kubectl -n $(NAMESPACE) delete secrets \
		central-db-password \
		central-db-tls \
		central-htpasswd \
		central-tls \
		proxy-config \
		scanner-db-password \
		scanner-db-tls \
		scanner-tls \
		|| true
	kubectl -n $(NAMESPACE) get secrets -o json        \
		| jq -r '.items[].metadata.name'               \
		| grep '^stackrox-generated'                   \
		| xargs kubectl -n $(NAMESPACE) delete secrets \
		|| true
	kubectl -n $(NAMESPACE) delete pvc --all || true

.PHONY: install-secured-cluster-services
install-secured-cluster-services: stackrox-secured-cluster-services-chart helm-values-secured-cluster-services.yaml
	@echo "Waiting for central to be ready..."
	kubectl -n $(NAMESPACE) rollout status deployment/central
	echo -n "Looking up central endpoint..."
	for i in $$(seq 120); do
		ip="$$(kubectl -n stackrox get service central-loadbalancer -o jsonpath='{.status.loadBalancer.ingress[0].ip}')"
		if [[ "$${ip}" != "" ]]; then
			CENTRAL_ENDPOINT="https://$${ip}:443"
			echo
			echo "Using central endpoint $$CENTRAL_ENDPOINT"
			break
		else
			echo -n "."
			sleep 1
		fi
	done
	if [[ -z "$$CENTRAL_ENDPOINT" ]]; then
		echo "No central endpoint found."
		exit 1
	fi
	helm install -n $(NAMESPACE) --set centralEndpoint="$$CENTRAL_ENDPOINT" \
		-f helm-values-secured-cluster-services.yaml \
		stackrox-secured-cluster-services ./stackrox-secured-cluster-services-chart

.PHONY: uninstall-secured-cluster-services
uninstall-secured-cluster-services:
	@helm uninstall -n $(NAMESPACE) stackrox-secured-cluster-services || true
	kubectl -n $(NAMESPACE) delete secrets helm-effective-cluster-name || true

.PHONY: install
install: install-central-services install-secured-cluster-services

.PHONY: uninstall
uninstall: uninstall-central-services uninstall-secured-cluster-services
