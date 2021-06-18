import React, { ReactElement, useState } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Button, PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';
import { actions as telemetryActions } from 'reducers/telemetryConfig';

import SystemConfigForm from './SystemConfigForm';
import Details from './Details';
import { PublicConfig, PrivateConfig, SystemConfig, TelemetryConfig } from './SystemConfigTypes';

export type PageProps = {
    systemConfig: SystemConfig;
    saveSystemConfig: (config) => void;
    telemetryConfig: TelemetryConfig;
    saveTelemetryConfig: (config) => void;
};

type InitialValues = {
    privateConfig: PrivateConfig;
    publicConfig: PublicConfig;
    telemetryConfig: TelemetryConfig;
};

function getInitialValues(
    systemConfig: SystemConfig,
    telemetryConfig: TelemetryConfig
): InitialValues {
    const modifiedSystemConfig = { ...systemConfig };
    modifiedSystemConfig.publicConfig = {
        header: {
            color: systemConfig?.publicConfig?.header?.color || '#000000',
            backgroundColor: systemConfig?.publicConfig?.header?.backgroundColor || '#FFFFFF',
            text: systemConfig?.publicConfig?.header?.text || '',
            enabled: systemConfig?.publicConfig?.header?.enabled || false,
            size: systemConfig?.publicConfig?.header?.size || 'UNSET',
        },
        footer: {
            color: systemConfig?.publicConfig?.footer?.color || '#000000',
            backgroundColor: systemConfig?.publicConfig?.footer?.backgroundColor || '#FFFFFF',
            text: systemConfig?.publicConfig?.footer?.text || '',
            enabled: systemConfig?.publicConfig?.footer?.enabled || false,
            size: systemConfig?.publicConfig?.footer?.size || 'UNSET',
        },
        loginNotice: {
            text: systemConfig?.publicConfig?.loginNotice?.text || '',
            enabled: systemConfig?.publicConfig?.loginNotice?.enabled || false,
        },
    };
    return {
        ...modifiedSystemConfig,
        telemetryConfig,
    };
}

const Page = ({
    systemConfig,
    saveSystemConfig,
    telemetryConfig,
    saveTelemetryConfig,
}: PageProps): ReactElement => {
    const [isEditing, setIsEditing] = useState(false);

    function editSystemConfig() {
        setIsEditing(true);
    }

    function cancelEdit() {
        setIsEditing(false);
    }

    function submitForm(config) {
        saveSystemConfig(config);
        saveTelemetryConfig(config.telemetryConfig);
        setIsEditing(false);
    }

    const initialValues = getInitialValues(systemConfig, telemetryConfig);

    return (
        <>
            <PageSection variant="light">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">System Configuration</Title>
                    </FlexItem>
                    <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                        <FlexItem>
                            {!isEditing && (
                                <Button
                                    variant="primary"
                                    data-testid="edit-btn"
                                    onClick={editSystemConfig}
                                >
                                    Edit
                                </Button>
                            )}
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <PageSection>
                {isEditing ? (
                    <SystemConfigForm
                        initialValues={initialValues}
                        onSubmitForm={submitForm}
                        onCancel={cancelEdit}
                    />
                ) : (
                    <Details systemConfig={systemConfig} telemetryConfig={telemetryConfig} />
                )}
            </PageSection>
        </>
    );
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
