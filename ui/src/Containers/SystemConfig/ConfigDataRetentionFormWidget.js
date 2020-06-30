/* eslint-disable jsx-a11y/label-has-associated-control */
import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import ReduxNumericInputField from 'Components/forms/ReduxNumericInputField';
import Widget from 'Components/Widget';

import { keyClassName } from './SystemConfig.constants';

const fieldLabelClassName = `${keyClassName} w-full capitalize items-center mr-4`;
const numericInputFieldClassName =
    'min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600';

const DayDurationFormField = ({ label, jsonPath, value }) => {
    const inputId = label.toLowerCase().replace(' ', '-');
    return (
        <div className="flex items-center my-4">
            <label className={fieldLabelClassName} htmlFor={`${inputId}-input`}>
                {label}:
            </label>
            <ReduxNumericInputField
                id={`${inputId}-input`}
                name={jsonPath}
                placeholder=""
                min={0}
                className={numericInputFieldClassName}
            />
            <span className="mx-2">{pluralize('day', value)}</span>
        </div>
    );
};

const DataRetentionFormWidget = ({ privateConfig }) => {
    return (
        <Widget id="data-retention-widget" header="Data Retention Configuration">
            <div className="px-4 md:flex w-full">
                <div className="flex-auto pr-0 md:pr-8 w-full md:w-1/2">
                    <DayDurationFormField
                        label="All Runtime Violations"
                        jsonPath="privateConfig.alertConfig.allRuntimeRetentionDurationDays"
                        value={privateConfig.alertConfig.allRuntimeRetentionDurationDays}
                    />
                    <DayDurationFormField
                        label="Runtime Violations For Deleted Deployments"
                        jsonPath="privateConfig.alertConfig.deletedRuntimeRetentionDurationDays"
                        value={privateConfig.alertConfig.deletedRuntimeRetentionDurationDays}
                    />
                </div>
                <div className="flex-auto pr-0 md:pr-8 w-full md:w-1/2">
                    <DayDurationFormField
                        label="Resolved Deploy-Phase Violations"
                        jsonPath="privateConfig.alertConfig.resolvedDeployRetentionDurationDays"
                        value={privateConfig.alertConfig.resolvedDeployRetentionDurationDays}
                    />
                    <DayDurationFormField
                        label="Images No Longer Deployed"
                        jsonPath="privateConfig.imageRetentionDurationDays"
                        value={privateConfig.imageRetentionDurationDays}
                    />
                </div>
            </div>
        </Widget>
    );
};

DataRetentionFormWidget.propTypes = {
    privateConfig: PropTypes.shape({
        alertConfig: PropTypes.shape({
            allRuntimeRetentionDurationDays: PropTypes.number,
            deletedRuntimeRetentionDurationDays: PropTypes.number,
            resolvedDeployRetentionDurationDays: PropTypes.number,
        }),
        imageRetentionDurationDays: PropTypes.number,
    }).isRequired,
};

export default DataRetentionFormWidget;
