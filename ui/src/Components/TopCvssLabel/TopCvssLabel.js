import React from 'react';
import PropTypes from 'prop-types';
import LabelChip from 'Components/LabelChip';
import { getSeverityChipType } from 'utils/vulnerabilityUtils';
import Tooltip from 'rc-tooltip';

const TopCvssLabel = ({ cvss, version, expanded }) => {
    if (!cvss && cvss !== 0) return 'N/A';

    const chipType = getSeverityChipType(cvss);
    const cvssNum = cvss.toFixed(1);
    const cvssText = expanded ? `Top CVSS: ${cvssNum}` : cvssNum || '';
    const extendedVersionText = `Scored using CVSS ${version}`;
    const versionText = expanded ? extendedVersionText : version;
    const labelElm = expanded ? (
        <LabelChip text={cvssText} type={chipType} />
    ) : (
        <Tooltip placement="bottom" overlay={<div>{extendedVersionText}</div>} mouseLeaveDelay={0}>
            <LabelChip text={cvssText} type={chipType} />
        </Tooltip>
    );
    return (
        <div className="mx-auto flex flex-col">
            {labelElm}
            <span className="pt-1 text-base-500 text-sm text-center">{versionText}</span>
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
