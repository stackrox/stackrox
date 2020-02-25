import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import merge from 'lodash/merge';

import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';
import { actions as telemetryActions } from 'reducers/telemetryConfig';

import PageHeader from 'Components/PageHeader';
import FormEditButtons from 'Components/FormEditButtons';
import Form from './Form';
import Detail from './Detail';

const defaultPublicConfig = {
    header: {
        color: '#000000',
        backgroundColor: '#ffffff'
    },
    footer: {
        color: '#000000',
        backgroundColor: '#ffffff'
    },
    loginNotice: null
};

const defaultPrivateConfig = {
    alertConfig: {
        allRuntimeRetentionDurationDays: 30,
        deletedRuntimeRetentionDurationDays: 7,
        resolvedDeployRetentionDurationDays: 7
    },
    imageRetentionDurationDays: 7
};

function getPrivateConfig(privateConfig = {}) {
    return merge(defaultPrivateConfig, privateConfig);
}

const Page = ({ systemConfig, saveSystemConfig, telemetryConfig, saveTelemetryConfig }) => {
    const [isEditing, setIsEditing] = useState(false);

    function saveHandler(config) {
        saveSystemConfig(config);
        saveTelemetryConfig(config.telemetryConfig);
        setIsEditing(false);
    }

    function getContent() {
        const modifiedSystemConfig = { ...systemConfig };
        if (!systemConfig.publicConfig) modifiedSystemConfig.publicConfig = defaultPublicConfig;
        const modifiedTelemetryConfig = { ...telemetryConfig };

        if (isEditing) {
            const safePrivateConfig = getPrivateConfig(systemConfig.privateConfig);
            const safeEditConfig = {
                ...modifiedSystemConfig,
                privateConfig: safePrivateConfig,
                telemetryConfig: modifiedTelemetryConfig
            };

            return (
                <Form
                    initialValues={safeEditConfig}
                    onSubmit={saveHandler}
                    config={safeEditConfig}
                />
            );
        }
        return <Detail config={modifiedSystemConfig} telemetryConfig={modifiedTelemetryConfig} />;
    }

    return (
        <section className="flex flex-1 flex-col h-full w-full">
            <div className="flex flex-1 flex-col w-full">
                <PageHeader header="System Configuration">
                    <div className="flex flex-1 justify-end">
                        <FormEditButtons
                            formName="system-config-form"
                            isEditing={isEditing}
                            setIsEditing={setIsEditing}
                        />
                    </div>
                </PageHeader>
                <div className="w-full h-full flex pb-0 bg-base-200 overflow-auto">
                    {getContent()}
                </div>
            </div>
        </section>
    );
};

Page.propTypes = {
    systemConfig: PropTypes.shape({
        publicConfig: PropTypes.shape({}),
        privateConfig: PropTypes.shape({})
    }),
    saveSystemConfig: PropTypes.func.isRequired,
    telemetryConfig: PropTypes.shape({
        enabled: PropTypes.bool
    }),
    saveTelemetryConfig: PropTypes.func.isRequired
};

Page.defaultProps = {
    systemConfig: {
        publicConfig: defaultPublicConfig,
        privateConfig: {}
    },
    telemetryConfig: {
        enabled: false
    }
};

const mapStateToProps = createStructuredSelector({
    systemConfig: selectors.getSystemConfig,
    telemetryConfig: selectors.getTelemetryConfig
});

const mapDispatchToProps = {
    saveSystemConfig: actions.saveSystemConfig,
    saveTelemetryConfig: telemetryActions.saveTelemetryConfig
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Page);
