import React from 'react';
import PropTypes from 'prop-types';
import uniq from 'lodash/uniq';

import Widget from 'Components/Widget';
import ComplianceStateLabel from 'Containers/Compliance/ComplianceStateLabel';

const processEvidence = data => {
    const evidence = data.map(evidenceResult => evidenceResult.message);
    return uniq(evidence).map(message => <li key={message}>{message}</li>);
};

const ControlAssessment = ({ className, controlResult }) => {
    if (!controlResult) return null;
    const controlState = controlResult.value.overallState;
    // eslint-disable-next-line
    const resourceType = controlResult.resource.__typename;
    const resourceName = controlResult.resource.name;
    const evidence = processEvidence(controlResult.value.evidence);
    return (
        <Widget
            header="Control Assessment"
            bodyClassName="flex-col"
            className={className}
            id="control-assessment"
        >
            <div className="flex flex-1">
                <div className="flex flex-col flex-1 justify-between items-start py-4 px-3">
                    <div className="pb-2">
                        <span className="font-700 pr-1">Control State:</span>
                        <span data-test-id="standard-name">
                            <ComplianceStateLabel state={controlState} />
                        </span>
                    </div>
                    <div>
                        <span className="font-700 pr-1">Time:</span>
                        <span data-test-id="control-name">-</span>
                    </div>
                </div>
                <div className="flex flex-col flex-1 justify-between items-start border-l border-base-300 py-4 px-3">
                    <div className="pb-2">
                        <span className="font-700 pr-1">{resourceType}:</span>
                        <span data-test-id="standard-name">{resourceName}</span>
                    </div>
                </div>
            </div>
            <div className="py-4 px-3 leading-loose whitespace-pre-wrap border-t border-base-300">
                <div className="font-700 pr-1">Evidence:</div>
                <ul className="pl-4">{evidence}</ul>
            </div>
        </Widget>
    );
};

ControlAssessment.propTypes = {
    className: PropTypes.string,
    controlResult: PropTypes.shape({})
};

ControlAssessment.defaultProps = {
    className: '',
    controlResult: null
};

export default ControlAssessment;
