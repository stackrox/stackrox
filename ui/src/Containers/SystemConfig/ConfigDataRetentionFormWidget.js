/* eslint-disable jsx-a11y/label-has-associated-control */
import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import ReduxNumericInputField from 'Components/forms/ReduxNumericInputField';

import { keyClassName } from './Page';

const DataRetentionFormWidget = ({ privateConfig }) => (
    <div className="bg-base-100 border-base-200 shadow" data-test-id="login-notice-config">
        <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center">
            Data Retention configuration
        </div>

        <div className="flex flex-col pt-2 pb-4 px-4 w-full">
            <div className="flex flex-wrap flex-col sm:flex-row w-full items-start justify-between">
                <div className="flex-auto pr-0 md:pr-8 w-full md:w-1/2">
                    <label className="flex flex-auto items-center w-full py-1">
                        <div className={`${keyClassName} flex w-full capitalize items-center`}>
                            All runtime violations
                        </div>
                        <ReduxNumericInputField
                            name="privateConfig.alertConfig.allRuntimeRetentionDurationDays"
                            placeholder=""
                            min={1}
                            className="min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600"
                        />
                        <span className="pl-1 w-12">
                            {pluralize(
                                'day',
                                privateConfig.alertConfig.allRuntimeRetentionDurationDays
                            )}
                        </span>
                    </label>
                    <label className="flex flex-auto items-center w-full py-1">
                        <div className={`${keyClassName} flex w-full capitalize items-center`}>
                            Runtime violations for deleted deployments
                        </div>
                        <ReduxNumericInputField
                            name="privateConfig.alertConfig.deletedRuntimeRetentionDurationDays"
                            placeholder=""
                            min={1}
                            className="min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600"
                        />
                        <span className="pl-1 w-12">
                            {pluralize(
                                'day',
                                privateConfig.alertConfig.deletedRuntimeRetentionDurationDays
                            )}
                        </span>
                    </label>
                </div>
                <div className="flex-auto pl-0 md:pl-8 w-full md:w-1/2">
                    <label className="flex flex-auto items-center w-full py-1">
                        <div className={`${keyClassName} flex w-full capitalize items-center`}>
                            Resolved deploy-phase violations
                        </div>
                        <ReduxNumericInputField
                            name="privateConfig.alertConfig.resolvedDeployRetentionDurationDays"
                            placeholder=""
                            min={1}
                            className="min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600"
                        />
                        <span className="pl-1 w-12">
                            {pluralize(
                                'day',
                                privateConfig.alertConfig.resolvedDeployRetentionDurationDays
                            )}
                        </span>
                    </label>
                    <label className="flex flex-auto items-center w-full py-1">
                        <div className={`${keyClassName} flex w-full capitalize items-center`}>
                            Images no longer deployed
                        </div>
                        <ReduxNumericInputField
                            name="privateConfig.imageRetentionDurationDays"
                            placeholder=""
                            min={1}
                            className="min-h-10 border-2 bg-base-100 border-base-300 text-base-600 p-3 rounded-r-sm cursor-pointer z-1 focus:border-base-300 w-12 font-600"
                        />
                        <span className="pl-1 w-12">
                            {pluralize('day', privateConfig.imageRetentionDurationDays)}
                        </span>
                    </label>
                </div>
            </div>
        </div>
    </div>
);

DataRetentionFormWidget.propTypes = {
    privateConfig: PropTypes.shape({
        alertConfig: PropTypes.shape({}),
        imageRetentionDurationDays: PropTypes.number
    }).isRequired
};

export default DataRetentionFormWidget;
