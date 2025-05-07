import React, { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import {
    Button,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Popover,
    Text,
    Title,
} from '@patternfly/react-core';

import { SystemConfig } from 'types/config.proto';
import { selectors } from 'reducers';

import { OutlinedQuestionCircleIcon } from '@patternfly/react-icons';
import PopoverBodyContent from 'Components/PopoverBodyContent';
import PrivateConfigDataRetentionDetails from './PrivateConfigDataRetentionDetails';
import PublicConfigBannerDetails from './PublicConfigBannerDetails';
import PublicConfigLoginDetails from './PublicConfigLoginDetails';
import PublicConfigTelemetryDetails from './PublicConfigTelemetryDetails';
import PlatformComponentsConfigDetails from './PlatformComponentsConfigDetails';

export type SystemConfigDetailsProps = {
    isClustersRoutePathRendered: boolean;
    systemConfig: SystemConfig;
};

function SystemConfigDetails({
    isClustersRoutePathRendered,
    systemConfig,
}: SystemConfigDetailsProps): ReactElement {
    const isTelemetryConfigured = useSelector(selectors.getIsTelemetryConfigured);
    return (
        <>
            <PageSection data-testid="platform-components-config">
                <Flex direction={{ default: 'row' }} spaceItems={{ default: 'spaceItemsMd' }}>
                    <Title headingLevel="h2" className="pf-v5-u-mb-md">
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
                            variant="plain"
                            isInline
                            aria-label="Show platform components config info"
                            className="pf-v5-u-p-0"
                        >
                            <OutlinedQuestionCircleIcon />
                        </Button>
                    </Popover>
                </Flex>
                <Text>
                    Define platform components using namespaces to segment platform security
                    findings from user workloads
                </Text>
                <div className="pf-v5-u-mt-lg">
                    <PlatformComponentsConfigDetails />
                </div>
            </PageSection>
            <PageSection data-testid="private-data-retention-config">
                <Title headingLevel="h2" className="pf-v5-u-mb-md">
                    Private data retention configuration
                </Title>
                <PrivateConfigDataRetentionDetails
                    isClustersRoutePathRendered={isClustersRoutePathRendered}
                    privateConfig={systemConfig?.privateConfig}
                />
            </PageSection>
            <PageSection data-testid="public-config">
                <Title headingLevel="h2" className="pf-v5-u-mb-md">
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
