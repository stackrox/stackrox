#!/usr/bin/env bash
set -eoux pipefail

previous_release=$1
release=$2

get_supported_versions() {
    supported_json="$(curl https://access.redhat.com/product-life-cycles/api/v1/products?name=Red%20Hat%20Advanced%20Cluster%20Security%20for%20Kubernetes)"
    
    supported_versions=($(echo $supported_json | jq -r '.data[0].versions[] | select(.type == "Full Support") | .name'))
    
    nversions=${#supported_versions[@]}
    for ((i = nversions - 1; i >= 0; i = i - 1)); do
        versions+=(${supported_versions[$i]})
    done
    
    versions+=("$release")
    nversions=${#versions[@]}
}


git clone git@gitlab.cee.redhat.com:cpaas-products/rhacs.git gitlab-rhacs
pushd gitlab-rhacs

git checkout -b setup-"$release"

cp -R versions/release-"$previous_release" versions/release-"$release"

pushd versions/release-"$release"

rm advisory_map.yml

# Update release.yml with the correct version. Sinple sed should be safe, but check changes
sed -i "s|$previous_release|$release|" release.yml

# Update product.yml with the correct version. 
# This is a little more complicated since not all occurances of the old release should be changed
# Check changes
sed -i "s|rhacs-$previous_release|rhacs-$release|" product.yml
sed -i "s|RHACS $previous_release|RHACS $release|" product.yml
sed -i "s|RHACS-$previous_release|RHACS-$release|" product.yml
sed -i "s|Kubernetes $previous_release|Kubernetes $release|" product.yml
yq w -i product.yml product.release.version "${release}".0 --style=single
yq w -i product.yml honeybadger.version "${release}" --style=single
yq w -i product.yml product.honeybadger.version "${release}" --style=single

get_supported_versions

popd

# Add the versions to content_stream_tags
for file in versions/release-*/product.yml; do
    yq w -i "$file" delivery-repo-content.content_stream_tags '' 
    for ((i = 0; i < nversions; i = i + 1)); do
            yq w -i "$file" delivery-repo-content.content_stream_tags[$i] ${versions[$[i]]} --style=double
    done
done
sed -i 's|!!merge ||' versions/release-*/product.yml

# TODO Automate this once there is confidance that this script is working.
echo "Commit and push the changes. Create an MR."
