import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

const NumberBox = ({ label, value, suffix }) => (
    <div className="min-h-32 w-full md:w-1/4 px-2 pb-4 md:pb-0">
        <div className="border border-base-400 rounded bg-tertiary-200 flex flex-col min-h-32 items-center py-1 justify-between">
            <div className="flex flex-col w-full border-b border-base-400 font-700 h-12 px-2 justify-center text-center capitalize leading-tight">
                <span>{label}</span>
            </div>
            <div className="flex flex-col justify-center flex-grow font-600 text-primary-700 text-5xl">
                {!value && `Never deleted`}
                {value > 0 && `${value} ${pluralize(suffix, value)}`}
            </div>
        </div>
    </div>
);

NumberBox.propTypes = {
    label: PropTypes.string.isRequired,
    value: PropTypes.number,
    suffix: PropTypes.string
};

NumberBox.defaultProps = {
    value: 0,
    suffix: ''
};

const DataRetentionDetailWidget = ({ config }) => {
    // safeguard, because on initial navigate, some nested objects are not loaded yet
    const privateConfig = config.privateConfig || {};
    const alertConfig = privateConfig.alertConfig || {};

    return (
        <div className="bg-base-100 border-base-200 shadow" data-test-id="login-notice-config">
            <div className="py-2 px-4 border-b border-base-300 text-base-600 font-700 text-lg capitalize flex justify-between items-center h-10">
                Data Retention Configuration{' '}
            </div>
            <div className="flex sm:flex-col md:flex-row flex-wrap px-2 sm:pb-2 py-4 w-full">
                <NumberBox
                    label="All runtime violations"
                    value={alertConfig.allRuntimeRetentionDurationDays}
                    suffix="Day"
                />
                <NumberBox
                    label="Runtime violations for deleted&nbsp;deployments"
                    value={alertConfig.deletedRuntimeRetentionDurationDays}
                    suffix="Day"
                />
                <NumberBox
                    label="Resolved deploy-phase violations"
                    value={alertConfig.resolvedDeployRetentionDurationDays}
                    suffix="Day"
                />
                <NumberBox
                    label="Images no longer deployed"
                    value={privateConfig.imageRetentionDurationDays}
                    suffix="Day"
                />
            </div>
        </div>
    );
};

DataRetentionDetailWidget.propTypes = {
    config: PropTypes.shape({
        publicConfig: PropTypes.shape({
            loginNotice: PropTypes.shape({})
        }),
        privateConfig: PropTypes.shape({})
    }).isRequired
};

export default DataRetentionDetailWidget;
