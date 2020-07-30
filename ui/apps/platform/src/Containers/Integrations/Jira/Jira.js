import React, { useState } from 'react';
import { Field } from 'redux-form';
import PropTypes from 'prop-types';

import ToggleSwitch from 'Components/ToggleSwitch';

const priorityRowClassName =
    'border-2 rounded p-2 bg-base-100 border-base-300 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10';

const defaultSeverities = [
    {
        severity: 'CRITICAL_SEVERITY',
        priorityName: 'P0-Highest',
    },
    {
        severity: 'HIGH_SEVERITY',
        priorityName: 'P1-High',
    },
    {
        severity: 'MEDIUM_SEVERITY',
        priorityName: 'P2-Medium',
    },
    {
        severity: 'LOW_SEVERITY',
        priorityName: 'P3-Low',
    },
];

const PriorityMapping = ({ fields }) => {
    const [usePriorityMapping, setUsePriorityMapping] = useState(false);

    const onClickHandler = () => () => {
        setUsePriorityMapping(!usePriorityMapping);
    };

    if (!fields.length) {
        defaultSeverities.forEach((severity) => fields.push(severity));
    }

    return (
        <div className="w-full">
            <div className="w-full text-right">
                <ToggleSwitch
                    id="priority-mapping-toggle"
                    toggleHandler={onClickHandler()}
                    enabled={usePriorityMapping}
                    small
                />
            </div>
            {usePriorityMapping &&
                fields.map((priorityItem) => (
                    <div
                        key={`${priorityItem.severity}--${priorityItem.priorityName}`}
                        className="w-full flex py-1"
                    >
                        <Field
                            key={`${priorityItem}.severity`}
                            name={`${priorityItem}.severity`}
                            component="input"
                            type="text"
                            className={`${priorityRowClassName} mr-1 w-1/3`}
                            placeholder="Severity"
                            disabled
                        />
                        <Field
                            key={`${priorityItem}.priorityName`}
                            name={`${priorityItem}.priorityName`}
                            component="input"
                            type="text"
                            className={`${priorityRowClassName} w-2/3`}
                            placeholder="Priority Name (e.g. P0-Highest)"
                        />
                    </div>
                ))}
        </div>
    );
};

PriorityMapping.propTypes = {
    fields: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

export default PriorityMapping;
