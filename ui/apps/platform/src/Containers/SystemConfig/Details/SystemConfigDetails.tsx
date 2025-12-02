import type { ReactElement } from 'react';
import {
    Button,
    Content,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Popover,
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
        <>
            <PageSection hasBodyWrapper={false} data-testid="platform-components-config">
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Title headingLevel="h2" className="pf-v6-u-mb-md">
                        Platform components configuration
                    </Title>
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
                            className="pf-v6-u-p-0"
                        />
                    </Popover>
                </Flex>
                <Content component="p">
                    Define platform components using namespaces to segment platform security
                    findings from user workloads
                </Content>
                <div className="pf-v6-u-mt-lg">
                    <PlatformComponentsConfigDetails
                        platformComponentConfig={systemConfig.platformComponentConfig}
                    />
                </div>
            </PageSection>
            <PageSection hasBodyWrapper={false} data-testid="private-data-retention-config">
                <Title headingLevel="h2" className="pf-v6-u-mb-md">
                    Private data retention configuration
                </Title>
                <PrivateConfigDataRetentionDetails
                    isClustersRoutePathRendered={isClustersRoutePathRendered}
                    privateConfig={systemConfig?.privateConfig}
                />
            </PageSection>
            <PageSection hasBodyWrapper={false} data-testid="private-prometheus-config">
                <Title headingLevel="h2" className="pf-v6-u-mb-md">
                    Prometheus metrics configuration
                </Title>
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
            </PageSection>
            <PageSection hasBodyWrapper={false} data-testid="public-config">
                <Title headingLevel="h2" className="pf-v6-u-mb-md">
                    Public configuration
                </Title>
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
            </PageSection>
        </>
    );
}

export default SystemConfigDetails;
