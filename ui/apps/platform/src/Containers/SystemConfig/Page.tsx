import React, { ReactElement, useEffect, useState } from 'react';
import { Button, PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import { actions } from 'reducers/systemConfig';
import { fetchSystemConfig, saveSystemConfig } from 'services/SystemConfigService';
import { fetchTelemetryConfig, saveTelemetryConfig } from 'services/TelemetryService';
import { SystemConfig } from 'types/config.proto';
import { TelemetryConfig } from 'types/telemetry.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import SystemConfigForm from './SystemConfigForm';
import Details from './Details';

const SystemConfigPage = (): ReactElement => {
    const [isEditing, setIsEditing] = useState(false);

    const [systemConfig, setSystemConfig] = useState<SystemConfig | null>(null);
    const [isLoadingSystemConfig, setIsLoadingSystemConfig] = useState(false);
    const [systemConfigErrorMessage, setSystemConfigErrorMessage] = useState('');

    const [telemetryConfig, setTelemetryConfig] = useState<TelemetryConfig | null>(null);
    const [isLoadingTelemetryConfig, setIsLoadingTelemetryConfig] = useState(false);
    const [telemetryConfigErrorMessage, setTelemetryConfigErrorMessage] = useState('');

    useEffect(() => {
        setIsLoadingSystemConfig(true);
        fetchSystemConfig()
            .then((data) => {
                setSystemConfig(data);
                setSystemConfigErrorMessage('');
            })
            .catch((error) => {
                setSystemConfig(null);
                setSystemConfigErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoadingSystemConfig(false));
    }, []);

    useEffect(() => {
        setIsLoadingTelemetryConfig(true);
        fetchTelemetryConfig()
            .then((data) => {
                setTelemetryConfig(data);
                setTelemetryConfigErrorMessage('');
            })
            .catch((error) => {
                setTelemetryConfig(null);
                setTelemetryConfigErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoadingTelemetryConfig(false));
    }, []);

    const isLoading = isLoadingSystemConfig || isLoadingTelemetryConfig;

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

export default SystemConfigPage;
