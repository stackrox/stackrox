import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';
import { actions as telemetryActions } from 'reducers/telemetryConfig';

import FormEditButtons from 'Components/FormEditButtons';
import Form from './Form';
import Details from './Details';

const defaultPublicConfig = {
    header: {
        color: '#000000',
        backgroundColor: '#ffffff',
    },
    footer: {
        color: '#000000',
        backgroundColor: '#ffffff',
    },
    loginNotice: null,
};

const Page = ({ systemConfig, saveSystemConfig, telemetryConfig, saveTelemetryConfig }) => {
    const [isEditing, setIsEditing] = useState(false);

    function saveHandler(config) {
        saveSystemConfig(config);
        saveTelemetryConfig(config.telemetryConfig);
        setIsEditing(false);
    }

    const modifiedSystemConfig = { ...systemConfig };
    if (!systemConfig.publicConfig) {
        modifiedSystemConfig.publicConfig = defaultPublicConfig;
    }
    const safePrivateConfig = systemConfig.privateConfig;
    const safeEditConfig = {
        ...modifiedSystemConfig,
        privateConfig: safePrivateConfig,
        telemetryConfig,
    };

    return (
        <>
            <PageSection variant="light">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">System Configuration</Title>
                    </FlexItem>
                    <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                        <FlexItem>
                            <FormEditButtons
                                formName="system-config-form"
                                isEditing={isEditing}
                                setIsEditing={setIsEditing}
                            />
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <PageSection>
                {isEditing ? (
                    <Form
                        initialValues={safeEditConfig}
                        onSubmit={saveHandler}
                        config={safeEditConfig}
                    />
                ) : (
                    <Details systemConfig={systemConfig} telemetryConfig={telemetryConfig} />
                )}
            </PageSection>
        </>
    );
};

Page.propTypes = {
    systemConfig: PropTypes.shape({
        publicConfig: PropTypes.shape({}),
        privateConfig: PropTypes.shape({}),
    }),
    saveSystemConfig: PropTypes.func.isRequired,
    telemetryConfig: PropTypes.shape({
        enabled: PropTypes.bool,
    }),
    saveTelemetryConfig: PropTypes.func.isRequired,
};

Page.defaultProps = {
    systemConfig: {
        publicConfig: defaultPublicConfig,
        privateConfig: {},
    },
    telemetryConfig: {
        enabled: false,
    },
};

const mapStateToProps = createStructuredSelector({
    systemConfig: selectors.getSystemConfig,
    telemetryConfig: selectors.getTelemetryConfig,
});

const mapDispatchToProps = {
    saveSystemConfig: actions.saveSystemConfig,
    saveTelemetryConfig: telemetryActions.saveTelemetryConfig,
};

export default connect(mapStateToProps, mapDispatchToProps)(Page);
