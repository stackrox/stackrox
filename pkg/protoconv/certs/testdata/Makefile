define ca_config
[ ca ]
default_ca = test_ca

[ test_ca ]
new_certs_dir = ./newcerts
database = ./ca-db.txt
serial = ./ca-serial.txt
private_key = ./ca-key.pem
certificate = ./ca-cert.pem
default_days = 3650
default_md = sha256
policy = ca_policy

[ ca_policy ]
countryName = optional
stateOrProvinceName = optional
localityName = optional
organizationName = optional
organizationalUnitName = optional
commonName = optional
emailAddress = optional
endef
export ca_config

.PHONY: clean
clean:
	rm -rf ./ca-key.pem ./ca-serial.txt ./ca-serial.txt* ./ca-db.txt* ./newcerts ./ca-config.txt *-csr.pem

.PHONY: setup
setup:
	echo "$${ca_config}" > ./ca-config.txt
	echo -n > ca-db.txt
	echo 01 > ./ca-serial.txt
	rm -rf newcerts
	mkdir -p newcerts

.PHONY: ca
ca: setup
	# Generate CA.
	openssl genpkey -algorithm RSA -out ca-key.pem
	openssl req -new -x509 -days 3650 -key ca-key.pem -sha256 -out ca-cert.pem -subj "/CN=StackRox Certificate Authority"

%-key.pem:
	openssl genpkey -algorithm RSA -out $@

%-csr.pem: %-key.pem
	openssl req -new -key $< -out $@ -subj "/O=f63cc867-44c7-401a-b5de-696cc9fb6629/OU=$(SERVICE_NAME)/CN=$(SERVICE_NAME): 00000000-0000-0000-0000-000000000000"

%-cert.pem: %-csr.pem
	openssl ca -config ./ca-config.txt -batch -notext -in $< -out $@

.PHONY: sensor
sensor: SERVICE_NAME=SENSOR_SERVICE
sensor: sensor-key.pem sensor-csr.pem sensor-cert.pem

.PHONY: collector
collector: SERVICE_NAME=COLLECTOR_SERVICE
collector: collector-key.pem collector-csr.pem collector-cert.pem

.PHONY: delete-existing
delete-existing:
	rm -f collector-cert.pem collector-key.pem sensor-cert.pem sensor-key.pem

.PHONY: add-test-annotation
add-test-annotation:
	for f in *-key.pem; do sed -i -e 's/PRIVATE KEY/PRIVATE TEST KEY/;' $$f; done

.PHONY: recreate
recreate: delete-existing ca collector sensor add-test-annotation clean
