#!/bin/bash

# Based on: https://gist.github.com/shreeve/3358901a26a21d4ddee0e1342be7749d
# See https://gist.github.com/fntlnz/cf14feb5a46b2eda428e000157447309

function generate() {
	name="Stackrox, Inc."
	base=$1
	root=$2

	echo "create root key and certificate"
	openssl genrsa -out "${root}.key" 3072
	openssl req -x509 -nodes -sha256 -new -key "${root}.key" -out "${root}.crt" -days 18500 \
		-subj "/CN=Custom Root" \
		-addext "keyUsage = critical, keyCertSign" \
		-addext "basicConstraints = critical, CA:TRUE, pathlen:0" \
		-addext "subjectKeyIdentifier = hash"

	echo "create our key and certificate signing request"
	openssl genrsa -out "${base}.key" 2048
	openssl req -sha256 -new -key "${base}.key" -out "${base}.csr" \
		-subj "/CN=*.${base}/O=${name}/OU=Stackrox QA" \
		-reqexts SAN -config <(echo -e "[dn]\nCN=localhost\n[req]\ndistinguished_name = dn\n[SAN]\nsubjectAltName=DNS:${base},DNS:*.${base},IP:127.0.0.1\n")

	echo "create our final certificate and sign it"
	openssl x509 -req -sha256 -in "${base}.csr" -out "${base}.crt" -days 18500 \
		-CAkey "${root}.key" -CA "${root}.crt" -CAcreateserial -extfile <(
			cat <<END
    subjectAltName = DNS:${base},DNS:*.${base},IP:127.0.0.1
    keyUsage = critical, digitalSignature, keyEncipherment
    extendedKeyUsage = serverAuth
    basicConstraints = CA:FALSE
    authorityKeyIdentifier = keyid:always
    subjectKeyIdentifier = none
END
		)

	echo "remove unused files"
	rm "${root}.key" "${root}.srl" "${base}.csr" "${base}.key" "${base}.csr"

	echo "review files"
	echo "--"
	openssl x509 -in "${root}.crt" -noout -text
	echo "--"
	openssl x509 -in "${base}.crt" -noout -text
	echo "--"
}

generate "self-signed.invalid" "unknown-root"
generate "untrusted-root.invalid" "root"

rm unknown-root.crt
