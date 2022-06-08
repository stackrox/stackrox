import React, { ReactElement, ReactNode, useEffect, useState } from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import { fetchSystemConfig } from 'services/SystemConfigService';
import { SystemConfig } from 'types/config.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import SystemConfigForm from './SystemConfigForm';
import Details from './Details';

const SystemConfigPage = (): ReactElement => {
    const [isEditing, setIsEditing] = useState(false);

    const [systemConfig, setSystemConfig] = useState<SystemConfig | null>(null);
    const [isLoading, setIsLoading] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        setIsLoading(true);
        fetchSystemConfig()
            .then((data) => {
                setSystemConfig(data);
                setErrorMessage('');
            })
            .catch((error) => {
                setSystemConfig(null);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, []);

    function onClickEdit() {
        setIsEditing(true);
    }

    function setIsNotEditing() {
        setIsEditing(false);
    }

    let content: ReactNode = null;

    if (isLoading) {
        content = (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    } else if (systemConfig) {
        content = isEditing ? (
            <SystemConfigForm
                systemConfig={systemConfig}
                setSystemConfig={setSystemConfig}
                setIsNotEditing={setIsNotEditing}
            />
        ) : (
            <Details systemConfig={systemConfig} />
        );
    } else {
        content = (
            <Alert variant="warning" isInline title="Failed to get system configuration">
                {errorMessage}
            </Alert>
        );
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
                                onClick={onClickEdit}
                            >
                                Edit
                            </Button>
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <PageSection>{content}</PageSection>
        </>
    );
};

export default SystemConfigPage;
