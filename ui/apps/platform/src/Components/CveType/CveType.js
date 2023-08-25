import React from 'react';
import PropTypes from 'prop-types';

const cveTypes = ['IMAGE_CVE', 'K8S_CVE', 'ISTIO_CVE', 'NODE_CVE', 'OPENSHIFT_CVE'];
const cveTypeMap = {
    IMAGE_CVE: 'Image CVE',
    K8S_CVE: 'Kubernetes CVE',
    ISTIO_CVE: 'Istio CVE',
    NODE_CVE: 'Node CVE',
    OPENSHIFT_CVE: 'OpenShift CVE',
};

const CveType = ({ types }) => {
    const sortedTypes = types.map((x) => cveTypeMap[x] || 'Unknown').sort();

    return (
        <span>
            <div className="flex flex-col">
                {sortedTypes.map((cveType) => (
                    <div className="flex justify-center" key={cveType}>
                        {cveType}
                    </div>
                ))}
            </div>
        </span>
    );
};

CveType.propTypes = {
    types: PropTypes.arrayOf(PropTypes.oneOf(cveTypes)),
};

CveType.defaultProps = {
    types: [],
};

export default CveType;
