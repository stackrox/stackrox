import React from 'react';
import PropTypes from 'prop-types';

import { standardLabels } from 'messages/standards';
import entityTypes from 'constants/entityTypes';
import NIST from 'images/nist.svg';
import PCI from 'images/pci.svg';
import HIPAA from 'images/hipaa.svg';
import CIS from 'images/cis.svg';

import Widget from 'Components/Widget';

const svgMapping = {
    [entityTypes.CIS_Kubernetes_v1_5]: CIS,
    [entityTypes.PCI_DSS_3_2]: PCI,
    [entityTypes.HIPAA_164]: HIPAA,
    [entityTypes.NIST_800_190]: NIST,
    [entityTypes.NIST_SP_800_53_Rev_4]: NIST,
};

const ControlDetails = ({ standardId, standardName, control, description, className }) => (
    <Widget
        header="Control details"
        bodyClassName="flex-col"
        className={className}
        id="control-details"
    >
        <div className="flex bg-tertiary-200 m-1">
            {svgMapping[standardId] && (
                <img src={svgMapping[standardId]} alt={standardId} className="h-18" />
            )}
            <div className="flex flex-col justify-center p-3">
                <div className="pb-2">
                    <span className="font-700 pr-1">Standard:</span>
                    <span data-testid="standard-name">
                        {standardLabels[standardId] || standardName}
                    </span>
                </div>
                <div>
                    <span className="font-700 pr-1">Control:</span>
                    <span data-testid="control-name">{control}</span>
                </div>
            </div>
        </div>
        <div className="px-4 py-3 leading-loose whitespace-pre-wrap">{description}</div>
    </Widget>
);

ControlDetails.propTypes = {
    standardId: PropTypes.string.isRequired,
    control: PropTypes.string.isRequired,
    description: PropTypes.string.isRequired,
    className: PropTypes.string,
    standardName: PropTypes.string,
};

ControlDetails.defaultProps = {
    className: '',
    standardName: '',
};

export default ControlDetails;
