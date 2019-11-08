import ReduxTextField from 'Components/forms/ReduxTextField';
import React from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxSelectCreatableField from 'Components/forms/ReduxSelectCreatableField';

const SingleScope = ({
    clusterOptions,
    fieldBasePath,
    isDeploymentScope,
    deploymentOptions,
    deleteScopeHandler
}) => {
    const actualBasePath = isDeploymentScope ? `${fieldBasePath}.scope` : fieldBasePath;
    return (
        <div key={actualBasePath} className="w-full pb-2">
            <ReduxSelectField
                name={`${actualBasePath}.cluster`}
                component="input"
                options={clusterOptions}
                type="text"
                className="border-2 rounded p-2 my-1 mr-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                placeholder="Cluster"
            />
            {isDeploymentScope && (
                <ReduxSelectCreatableField
                    name={`${fieldBasePath}.name`}
                    component="input"
                    options={deploymentOptions}
                    type="text"
                    className="border-2 rounded p-2 my-1 mr-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                    placeholder="Deployment Name"
                />
            )}
            <ReduxTextField
                name={`${actualBasePath}.namespace`}
                component="input"
                type="text"
                className="border-2 rounded p-2 my-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                placeholder="Namespace"
            />
            <div className="flex">
                <ReduxTextField
                    name={`${actualBasePath}.label.key`}
                    component="input"
                    type="text"
                    className="border-2 rounded p-2 my-1 mr-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                    placeholder="Label Key"
                />
                <ReduxTextField
                    name={`${actualBasePath}.label.value`}
                    component="input"
                    type="text"
                    className="border-2 rounded p-2 my-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                    placeholder="Label Value"
                />
                <button
                    className="ml-2 p-2 my-1 flex rounded-r-sm text-base-100 uppercase text-center text-alert-700 hover:text-alert-800 bg-alert-200 hover:bg-alert-300 border-2 border-alert-300 items-center rounded"
                    onClick={deleteScopeHandler}
                    type="button"
                >
                    <Icon.X size="20" />
                </button>
            </div>
        </div>
    );
};

SingleScope.propTypes = {
    clusterOptions: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired
        })
    ).isRequired,
    deploymentOptions: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired
        })
    ).isRequired,
    fieldBasePath: PropTypes.string.isRequired,
    isDeploymentScope: PropTypes.bool.isRequired,
    deleteScopeHandler: PropTypes.func.isRequired
};

export default SingleScope;
