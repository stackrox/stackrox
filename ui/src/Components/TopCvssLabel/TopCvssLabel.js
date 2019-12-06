import React from 'react';
import PropTypes from 'prop-types';
import LabelChip from 'Components/LabelChip';
import { getSeverityChipType } from 'utils/vulnerabilityUtils';
import Tooltip from 'rc-tooltip';

const CVSSLabelChip = ({ cvss, expanded }) => {
    const chipType = getSeverityChipType(cvss);
    const cvssNum = cvss.toFixed(1);
    const cvssText = expanded ? `Top CVSS: ${cvssNum}` : cvssNum || '';
    return <LabelChip text={cvssText} type={chipType} size="large" />;
};

CVSSLabelChip.propTypes = {
    cvss: PropTypes.number.isRequired,
    expanded: PropTypes.bool.isRequired
};

const TopCvssLabel = ({ cvss, version, expanded }) => {
    if (!cvss && cvss !== 0) return 'N/A';

    const extendedVersionText = `Scored using CVSS ${version}`;
    const versionText = expanded ? extendedVersionText : version;

    const labelElm = expanded ? (
        <CVSSLabelChip cvss={cvss} expanded={expanded} />
    ) : (
        <Tooltip placement="bottom" overlay={<div>{extendedVersionText}</div>} mouseLeaveDelay={0}>
            <CVSSLabelChip cvss={cvss} expanded={expanded} />
        </Tooltip>
    );
    return (
        <div className="mx-auto flex flex-col">
            {labelElm}
            <span className="pt-1 text-base-500 text-xs font-700 text-center">
                {' '}
                ({versionText})
            </span>
        </div>
    );
};

TopCvssLabel.propTypes = {
    cvss: PropTypes.number.isRequired,
    version: PropTypes.string.isRequired,
    expanded: PropTypes.bool
};

TopCvssLabel.defaultProps = {
    expanded: false
};

export default TopCvssLabel;
