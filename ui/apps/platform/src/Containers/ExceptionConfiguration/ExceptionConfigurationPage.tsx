import React from 'react';
import { Bullseye, PageSection, Spinner, Tab, Tabs, Title } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { useVulnerabilitiesExceptionConfig } from './useVulnerabilitiesExceptionConfig';
import VulnerabilitiesConfiguration from './VulnerabilitiesConfiguration';

const exceptionConfigurationCategories = ['Vulnerabilities'] as const;

function ExceptionConfigurationPage() {
    const [category, setCategory] = useURLStringUnion('category', exceptionConfigurationCategories);

    const vulnerabilitiesConfigRequest = useVulnerabilitiesExceptionConfig();
    const { config, isConfigLoading, isUpdateInProgress, configLoadError, updateConfig } =
        vulnerabilitiesConfigRequest;

    return (
        <>
            <PageTitle title="Exception configuration" />
            <PageSection variant="light">
                <Title headingLevel="h1">Exception configuration</Title>
            </PageSection>
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Tabs
                    activeKey={category}
                    onSelect={(e, value) => setCategory(value)}
                    usePageInsets
                    mountOnEnter
                >
                    <Tab eventKey="Vulnerabilities" title="Vulnerabilities">
                        {isConfigLoading && !config && (
                            <Bullseye>
                                <Spinner aria-label="Loading current vulnerability exception configuration" />
                            </Bullseye>
                        )}
                        {configLoadError && (
                            <Bullseye>
                                <EmptyStateTemplate
                                    title="Error loading vulnerability exception configuration"
                                    headingLevel="h2"
                                    icon={ExclamationCircleIcon}
                                    iconClassName="pf-u-danger-color-100"
                                >
                                    {getAxiosErrorMessage(configLoadError)}
                                </EmptyStateTemplate>
                            </Bullseye>
                        )}
                        {config && (
                            <VulnerabilitiesConfiguration
                                config={config}
                                isUpdateInProgress={isUpdateInProgress}
                                updateConfig={updateConfig}
                            />
                        )}
                    </Tab>
                </Tabs>
            </PageSection>
        </>
    );
}

export default ExceptionConfigurationPage;
