import React from 'react';
import PropTypes from 'prop-types';

import LabelChip from 'Components/LabelChip';

const cveTypes = ['IMAGE_CVE', 'K8S_CVE', 'ISTIO_CVE'];
const cveTypeMap = {
    IMAGE_CVE: 'Image CVE',
    K8S_CVE: 'Kubernetes CVE',
    ISTIO_CVE: 'Istio CVE',
};

const CveType = ({ type, context }) => {
    const typeText = cveTypeMap[type] || 'Unknown';

    return context === 'callout' ? (
        <LabelChip type="base" text={`Type: ${typeText}`} />
    ) : (
        <span>{typeText}</span>
    );
};

CveType.propTypes = {
    type: PropTypes.oneOf(cveTypes).isRequired,
    context: PropTypes.oneOf(['callout', 'bare']),
};

CveType.defaultProps = {
    context: 'bare',
};

export default CveType;
