import React from 'react';
import PropTypes from 'prop-types';

import LabelChip from 'Components/LabelChip';

const cveTypes = ['IMAGE_CVE', 'K8S_CVE', 'ISTIO_CVE', 'NODE_CVE'];
const cveTypeMap = {
    IMAGE_CVE: 'Image CVE',
    K8S_CVE: 'Kubernetes CVE',
    ISTIO_CVE: 'Istio CVE',
    NODE_CVE: 'Node CVE',
};

const CveType = ({ types, context }) => {
    const sortedTypes = types.map((x) => cveTypeMap[x] || 'Unknown').sort();

    return context === 'callout' ? (
        <LabelChip type="base" text={`Type: ${sortedTypes.join(', ')}`} />
    ) : (
        <span>
            <div className="flex flex-col">
                {sortedTypes.map((cveType) => (
                    // eslint-disable-next-line react/jsx-key
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
    context: PropTypes.oneOf(['callout', 'bare']),
};

CveType.defaultProps = {
    types: [],
    context: 'bare',
};

export default CveType;
