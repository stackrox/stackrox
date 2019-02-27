import { connect } from 'react-redux';
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
    [entityTypes.CIS_Docker_v1_1_0]: CIS,
    [entityTypes.CIS_Kubernetes_v1_2_0]: CIS,
    [entityTypes.PCI_DSS_3_2]: PCI,
    [entityTypes.HIPAA_164]: HIPAA,
    [entityTypes.NIST_800_190]: NIST
};

const ControlDetails = ({ standardId, control, description, className }) => (
    <Widget header="Control details" bodyClassName="flex-col" className={className}>
        <div className="flex bg-tertiary-200 m-1">
            <img src={svgMapping[standardId]} alt={standardId} className="h-18" />
            <div className="flex flex-col justify-center pl-3">
                <div className="pb-2">
                    <span className="font-700 pr-1">Standard:</span>
                    <span data-test-id="standard-name">{standardLabels[standardId]}</span>
                </div>
                <div>
                    <span className="font-700 pr-1">Control:</span>
                    <span data-test-id="control-name">{control}</span>
                </div>
            </div>
        </div>
        <div className="px-4 py-3 leading-loose">{description}</div>
    </Widget>
);

ControlDetails.propTypes = {
    standardId: PropTypes.string.isRequired,
    control: PropTypes.string.isRequired,
    description: PropTypes.string.isRequired,
    className: PropTypes.string
};

ControlDetails.defaultProps = {
    className: ''
};

export default connect()(ControlDetails);
