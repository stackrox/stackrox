import React, { useEffect } from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { formValueSelector, change } from 'redux-form';

import useFeatureFlagEnabled from 'hooks/useFeatureFlagEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import ReduxTextField from 'Components/forms/ReduxTextField';
import ReduxSelectField from 'Components/forms/ReduxSelectField';
import ReduxSelectCreatableField from 'Components/forms/ReduxSelectCreatableField';

const SingleScope = ({
    clusterOptions,
    fieldBasePath,
    isDeploymentScope,
    deploymentOptions,
    deleteScopeHandler,
    includesAuditLogEventSource,
    changeForm,
}) => {
    const auditLogEnabled = useFeatureFlagEnabled(knownBackendFlags.ROX_K8S_AUDIT_LOG_DETECTION);
    const actualBasePath = isDeploymentScope ? `${fieldBasePath}.scope` : fieldBasePath;
    const isAuditLogEventSource = auditLogEnabled && includesAuditLogEventSource;
    useEffect(() => {
        // clear Label key and value when Audit Log Event Source is selected
        if (isAuditLogEventSource) {
            changeForm(`${actualBasePath}.label`, undefined);
            if (isDeploymentScope) {
                changeForm(`${fieldBasePath}.name`, undefined);
            }
        }
    }, [changeForm, isAuditLogEventSource, actualBasePath, fieldBasePath, isDeploymentScope]);
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
                    disabled={isAuditLogEventSource}
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
                    disabled={isAuditLogEventSource}
                />
                <ReduxTextField
                    name={`${actualBasePath}.label.value`}
                    component="input"
                    type="text"
                    className="border-2 rounded p-2 my-1 border-base-300 w-1/2 font-600 text-base-600 hover:border-base-400 leading-normal min-h-10"
                    placeholder="Label Value"
                    disabled={isAuditLogEventSource}
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
            value: PropTypes.string.isRequired,
        })
    ).isRequired,
    deploymentOptions: PropTypes.arrayOf(
        PropTypes.shape({
            label: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired,
        })
    ).isRequired,
    fieldBasePath: PropTypes.string.isRequired,
    isDeploymentScope: PropTypes.bool.isRequired,
    deleteScopeHandler: PropTypes.func.isRequired,
    includesAuditLogEventSource: PropTypes.bool.isRequired,
    changeForm: PropTypes.func.isRequired,
};

const mapStateToProps = createStructuredSelector({
    includesAuditLogEventSource: (state) => {
        const eventSourceValue = formValueSelector('policyCreationForm')(state, 'eventSource');
        return eventSourceValue === 'AUDIT_LOG_EVENT';
    },
});

const mapDispatchToProps = (dispatch) => ({
    changeForm: (field, value) => dispatch(change('policyCreationForm', field, value)),
});

export default connect(mapStateToProps, mapDispatchToProps)(SingleScope);
