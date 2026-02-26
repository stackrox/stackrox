import type { ReactElement } from 'react';
import {
    Button,
    Content,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Popover,
    Stack,
    Title,
} from '@patternfly/react-core';
import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';

import PopoverBodyContent from 'Components/PopoverBodyContent';
import useTelemetryConfig from 'hooks/useTelemetryConfig';
import type { SystemConfig } from 'types/config.proto';

import PrivateConfigDataRetentionDetails from './PrivateConfigDataRetentionDetails';
import PublicConfigBannerDetails from './PublicConfigBannerDetails';
import PublicConfigLoginDetails from './PublicConfigLoginDetails';
import PublicConfigTelemetryDetails from './PublicConfigTelemetryDetails';
import PlatformComponentsConfigDetails from './PlatformComponentsConfigDetails';
import PrivateConfigPrometheusMetricsDetails from './PrivateConfigPrometheusMetricsDetails';

export type SystemConfigDetailsProps = {
    systemConfig: SystemConfig;
    isClustersRoutePathRendered: boolean;
};

function SystemConfigDetails({
    systemConfig,
    isClustersRoutePathRendered,
}: SystemConfigDetailsProps): ReactElement {
    const { isTelemetryConfigured } = useTelemetryConfig();

    return (
        <Stack hasGutter>
            <PageSection data-testid="platform-components-config">
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Title headingLevel="h2">Platform components configuration</Title>
                    <Popover
                        aria-label="Platform components config info"
                        bodyContent={
                            <PopoverBodyContent
                                headerContent="What is a platform component?"
                                bodyContent="Platform components include the underlying infrastructure, operators, and third-party services that support application development. Defining these components allow for categorization of security findings and segments them by area of responsibility."
                            />
                        }
                    >
                        <Button
                            icon={<OutlinedQuestionCircleIcon />}
                            variant="plain"
                            isInline
                            aria-label="Show platform components config info"
                        />
                    </Popover>
                </Flex>
                <Content component="p">
                    Define platform components using namespaces to segment platform security
                    findings from user workloads
                </Content>
                <PlatformComponentsConfigDetails
                    platformComponentConfig={systemConfig.platformComponentConfig}
                />
            </PageSection>
            <PageSection data-testid="private-data-retention-config">
                <Stack hasGutter>
                    <Title headingLevel="h2">Private data retention configuration</Title>
                    <PrivateConfigDataRetentionDetails
                        isClustersRoutePathRendered={isClustersRoutePathRendered}
                        privateConfig={systemConfig?.privateConfig}
                    />
                </Stack>
            </PageSection>
            <PageSection data-testid="private-prometheus-config">
                <Stack hasGutter>
                    <Title headingLevel="h2">Prometheus metrics configuration</Title>
                    <Content component="p">
                        The following Prometheus metrics are exposed on the API endpoint at the{' '}
                        <code>/metrics</code> path. Scrape requests require permissions to view
                        Administration resources and are subject for the scoped access control.
                    </Content>
                    <Grid hasGutter>
                        <PrivateConfigPrometheusMetricsDetails
                            privateConfig={systemConfig?.privateConfig}
                        />
                    </Grid>
                </Stack>
            </PageSection>
            <PageSection data-testid="public-config">
                <Stack hasGutter>
                    <Title headingLevel="h2">Public configuration</Title>
                    <Grid hasGutter>
                        <GridItem sm={12} md={6}>
                            <PublicConfigBannerDetails
                                type="header"
                                publicConfig={systemConfig?.publicConfig}
                            />
                        </GridItem>
                        <GridItem sm={12} md={6}>
                            <PublicConfigBannerDetails
                                type="footer"
                                publicConfig={systemConfig?.publicConfig}
                            />
                        </GridItem>
                        <GridItem sm={12} md={6}>
                            <PublicConfigLoginDetails publicConfig={systemConfig?.publicConfig} />
                        </GridItem>
                        {isTelemetryConfigured && (
                            <GridItem sm={12} md={6}>
                                <PublicConfigTelemetryDetails
                                    publicConfig={systemConfig?.publicConfig}
                                />
                            </GridItem>
                        )}
                    </Grid>
                </Stack>
            </PageSection>
        </Stack>
    );
}

export default SystemConfigDetails;
