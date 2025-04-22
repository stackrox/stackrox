import React from 'react';
import PropTypes from 'prop-types';

import { standardLabels } from 'messages/standards';

import Widget from 'Components/Widget';

const ControlDetails = ({ standardId, standardName, control, description, className }) => (
    <Widget
        header="Control details"
        bodyClassName="flex-col"
        className={className}
        id="control-details"
    >
        <div className="flex flex-col justify-center p-4">
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
        <div className="px-4 pb-4 leading-loose whitespace-pre-wrap">{description}</div>
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
