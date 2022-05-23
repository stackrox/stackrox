import React, { ReactElement, useState } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Button, PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions } from 'reducers/systemConfig';
import { actions as telemetryActions } from 'reducers/telemetryConfig';
import { SystemConfig } from 'types/config.proto';
import { TelemetryConfig } from 'types/telemetry.proto';

import SystemConfigForm from './SystemConfigForm';
import Details from './Details';

export type PageProps = {
    systemConfig: SystemConfig;
    saveSystemConfig: (systemConfig: SystemConfig) => void;
    telemetryConfig: TelemetryConfig;
    saveTelemetryConfig: (telemetryConfig: TelemetryConfig) => void;
};

const Page = ({
    systemConfig,
    saveSystemConfig,
    telemetryConfig,
    saveTelemetryConfig,
}: PageProps): ReactElement => {
    const [isEditing, setIsEditing] = useState(false);
    // TODO next step will call fetch functions directly from services instead of indirectly via sagas
    // Wait while either object is empty, which is initial state of the reducers.
    const isLoading =
        Object.keys(systemConfig).length === 0 || Object.keys(telemetryConfig).length === 0;

    function editSystemConfig() {
        setIsEditing(true);
    }

    function cancelEdit() {
        setIsEditing(false);
    }

    function onSubmit(systemConfigSubmitted, telemetryConfigSubmitted) {
        // TODO next step will receive the responses from requests in the form (so it can render error and loading).
        saveSystemConfig(systemConfigSubmitted);
        saveTelemetryConfig(telemetryConfigSubmitted);
        setIsEditing(false);
    }

    return (
        <>
            <PageSection variant="light" sticky="top">
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">System Configuration</Title>
                    </FlexItem>
                    <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                        <FlexItem>
                            <Button
                                variant="primary"
                                isDisabled={isEditing || isLoading}
                                onClick={editSystemConfig}
                            >
                                Edit
                            </Button>
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <PageSection>
                {isEditing ? (
                    <SystemConfigForm
                        systemConfig={systemConfig}
                        telemetryConfig={telemetryConfig}
                        onCancel={cancelEdit}
                        onSubmit={onSubmit}
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
