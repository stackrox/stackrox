import React from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    Label,
    PageSection,
    Text,
    Title,
} from '@patternfly/react-core';
import { CheckCircleIcon, InProgressIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import DateDistance from 'Components/DateDistance';
import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { vulnerabilitySeverityLabels } from 'messages/common';
import type { BaseImage, ScanningStatus } from '../types';

type BaseImageHeaderProps = {
    baseImage: BaseImage;
};

function getScanningStatusIcon(status: ScanningStatus) {
    switch (status) {
        case 'COMPLETED':
            return <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />;
        case 'IN_PROGRESS':
            return <InProgressIcon color="var(--pf-v5-global--info-color--100)" />;
        case 'FAILED':
            return <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />;
        default:
            return null;
    }
}

function getScanningStatusLabel(status: ScanningStatus) {
    switch (status) {
        case 'COMPLETED':
            return (
                <Label color="green" icon={getScanningStatusIcon(status)} isCompact>
                    Completed
                </Label>
            );
        case 'IN_PROGRESS':
            return (
                <Label color="blue" icon={getScanningStatusIcon(status)} isCompact>
                    In Progress
                </Label>
            );
        case 'FAILED':
            return (
                <Label color="red" icon={getScanningStatusIcon(status)} isCompact>
                    Failed
                </Label>
            );
        default:
            return null;
    }
}

/**
 * Header section for base image detail page showing name, status, and summary metrics
 */
function BaseImageHeader({ baseImage }: BaseImageHeaderProps) {
    const CriticalIcon = SeverityIcons.CRITICAL_VULNERABILITY_SEVERITY;
    const ImportantIcon = SeverityIcons.IMPORTANT_VULNERABILITY_SEVERITY;
    const ModerateIcon = SeverityIcons.MODERATE_VULNERABILITY_SEVERITY;
    const LowIcon = SeverityIcons.LOW_VULNERABILITY_SEVERITY;

    return (
        <>
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column' }}
                    spaceItems={{ default: 'spaceItemsSm' }}
                    alignItems={{ default: 'alignItemsStretch' }}
                >
                    <FlexItem>
                        <Title headingLevel="h1">{baseImage.name}</Title>
                    </FlexItem>
                    <FlexItem>
                        <Text component="small" className="pf-v5-u-color-200">
                            {baseImage.normalizedName}
                        </Text>
                    </FlexItem>
                    <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                        <FlexItem>{getScanningStatusLabel(baseImage.scanningStatus)}</FlexItem>
                        <FlexItem>
                            <Text component="small">
                                {baseImage.lastScanned ? (
                                    <>
                                        Last scanned <DateDistance date={baseImage.lastScanned} />
                                    </>
                                ) : (
                                    'Not scanned yet'
                                )}
                            </Text>
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>

            <PageSection>
                <Grid hasGutter md={4}>
                    <GridItem>
                        <Card isCompact isFlat isFullHeight>
                            <CardTitle>Total CVEs</CardTitle>
                            <CardBody>
                                <Flex
                                    direction={{ default: 'column' }}
                                    spaceItems={{ default: 'spaceItemsSm' }}
                                >
                                    <FlexItem>
                                        <Text className="pf-v5-u-font-size-2xl pf-v5-u-font-weight-bold">
                                            {baseImage.cveCount.total}
                                        </Text>
                                    </FlexItem>
                                    <Grid className="pf-v5-u-pl-sm">
                                        <GridItem span={6}>
                                            <Flex
                                                className="pf-v5-u-pt-sm"
                                                spaceItems={{ default: 'spaceItemsSm' }}
                                                alignItems={{ default: 'alignItemsCenter' }}
                                            >
                                                <CriticalIcon
                                                    title={
                                                        vulnerabilitySeverityLabels.CRITICAL_VULNERABILITY_SEVERITY
                                                    }
                                                />
                                                <Text>
                                                    {baseImage.cveCount.critical}{' '}
                                                    {
                                                        vulnerabilitySeverityLabels.CRITICAL_VULNERABILITY_SEVERITY
                                                    }
                                                </Text>
                                            </Flex>
                                        </GridItem>
                                        <GridItem span={6}>
                                            <Flex
                                                className="pf-v5-u-pt-sm"
                                                spaceItems={{ default: 'spaceItemsSm' }}
                                                alignItems={{ default: 'alignItemsCenter' }}
                                            >
                                                <ImportantIcon
                                                    title={
                                                        vulnerabilitySeverityLabels.IMPORTANT_VULNERABILITY_SEVERITY
                                                    }
                                                />
                                                <Text>
                                                    {baseImage.cveCount.high}{' '}
                                                    {
                                                        vulnerabilitySeverityLabels.IMPORTANT_VULNERABILITY_SEVERITY
                                                    }
                                                </Text>
                                            </Flex>
                                        </GridItem>
                                        <GridItem span={6}>
                                            <Flex
                                                className="pf-v5-u-pt-sm"
                                                spaceItems={{ default: 'spaceItemsSm' }}
                                                alignItems={{ default: 'alignItemsCenter' }}
                                            >
                                                <ModerateIcon
                                                    title={
                                                        vulnerabilitySeverityLabels.MODERATE_VULNERABILITY_SEVERITY
                                                    }
                                                />
                                                <Text>
                                                    {baseImage.cveCount.medium}{' '}
                                                    {
                                                        vulnerabilitySeverityLabels.MODERATE_VULNERABILITY_SEVERITY
                                                    }
                                                </Text>
                                            </Flex>
                                        </GridItem>
                                        <GridItem span={6}>
                                            <Flex
                                                className="pf-v5-u-pt-sm"
                                                spaceItems={{ default: 'spaceItemsSm' }}
                                                alignItems={{ default: 'alignItemsCenter' }}
                                            >
                                                <LowIcon
                                                    title={
                                                        vulnerabilitySeverityLabels.LOW_VULNERABILITY_SEVERITY
                                                    }
                                                />
                                                <Text>
                                                    {baseImage.cveCount.low}{' '}
                                                    {
                                                        vulnerabilitySeverityLabels.LOW_VULNERABILITY_SEVERITY
                                                    }
                                                </Text>
                                            </Flex>
                                        </GridItem>
                                    </Grid>
                                </Flex>
                            </CardBody>
                        </Card>
                    </GridItem>

                    <GridItem>
                        <Card isCompact isFlat isFullHeight>
                            <CardTitle>Images Using This Base</CardTitle>
                            <CardBody>
                                <Text className="pf-v5-u-font-size-2xl pf-v5-u-font-weight-bold">
                                    {baseImage.imageCount}
                                </Text>
                                <Text component="small" className="pf-v5-u-color-200">
                                    Application images built on this base
                                </Text>
                            </CardBody>
                        </Card>
                    </GridItem>

                    <GridItem>
                        <Card isCompact isFlat isFullHeight>
                            <CardTitle>Deployments Affected</CardTitle>
                            <CardBody>
                                <Text className="pf-v5-u-font-size-2xl pf-v5-u-font-weight-bold">
                                    {baseImage.deploymentCount}
                                </Text>
                                <Text component="small" className="pf-v5-u-color-200">
                                    Deployments running images with this base
                                </Text>
                            </CardBody>
                        </Card>
                    </GridItem>
                </Grid>
            </PageSection>
        </>
    );
}

export default BaseImageHeader;
