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
	rm -rf ca-serial.txt ca-serial.txt* ca-db.txt* newcerts ca-config.txt

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

.PHONY: add-test-annotation
add-test-annotation:
	for f in *-key.pem; do sed -i -e 's/PRIVATE KEY/PRIVATE TEST KEY/;' $$f; done

.PHONY: recreate
recreate: ca add-test-annotation clean
