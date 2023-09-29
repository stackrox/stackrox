import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import {
    Button,
    Card,
    CardBody,
    CardTitle,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    Popover,
    Title,
} from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import ClusterLabelsTable from 'Containers/Clusters/ClusterLabelsTable';
import { PrivateConfig } from 'types/config.proto';
import { clustersBasePath } from 'routePaths';

import { HelpIcon } from '@patternfly/react-icons';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { convertBetweenBytesAndMB } from '../SystemConfig.utils';

type DataRetentionValueProps = {
    value: number | undefined;
    suffix: string;
    shouldPluralize?: boolean;
    canRetainForever?: boolean;
};

function DataRetentionValue({
    value,
    suffix,
    canRetainForever = true,
    shouldPluralize = true,
}: DataRetentionValueProps): ReactElement {
    let content = 'Unknown';

    if (typeof value === 'number') {
        if (value === 0) {
            content = canRetainForever ? 'Never deleted' : 'Deleted in every pruning cycle';
        } else if (value > 0) {
            content = `${value} ${shouldPluralize ? pluralize(suffix, value) : suffix}`;
        }
    }

    return <span className="pf-u-font-size-xl pf-u-font-weight-bold">{content}</span>;
}

export type PrivateConfigDataRetentionDetailsProps = {
    isClustersRoutePathRendered: boolean;
    privateConfig: PrivateConfig;
};

const PrivateConfigDataRetentionDetails = ({
    isClustersRoutePathRendered,
    privateConfig,
}: PrivateConfigDataRetentionDetailsProps): ReactElement => {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    return (
        <Grid hasGutter md={6}>
            <GridItem>
                <Card isFlat className="pf-u-h-100">
                    <CardTitle>All runtime violations</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={privateConfig?.alertConfig?.allRuntimeRetentionDurationDays}
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat className="pf-u-h-100">
                    <CardTitle>Runtime violations for deleted deployments</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={privateConfig?.alertConfig?.deletedRuntimeRetentionDurationDays}
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat className="pf-u-h-100">
                    <CardTitle>Resolved deploy-phase violations</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={privateConfig?.alertConfig?.resolvedDeployRetentionDurationDays}
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat className="pf-u-h-100">
                    <CardTitle>Attempted deploy-phase violations</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={privateConfig?.alertConfig?.attemptedDeployRetentionDurationDays}
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat className="pf-u-h-100">
                    <CardTitle>Attempted runtime violations</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={
                                privateConfig?.alertConfig?.attemptedRuntimeRetentionDurationDays
                            }
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat className="pf-u-h-100">
                    <CardTitle>Images no longer deployed</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={privateConfig?.imageRetentionDurationDays}
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat>
                    <CardTitle>Expired vulnerability requests</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={privateConfig?.expiredVulnReqRetentionDurationDays}
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat>
                    <CardTitle>
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                            <FlexItem>Vulnerability report job history retention</FlexItem>
                            <FlexItem>
                                <Popover
                                    aria-label="Vulnerability report job history retention help text"
                                    bodyContent={
                                        <div>
                                            <p>
                                                The{' '}
                                                <strong>
                                                    &quot;Vulnerability report job history
                                                    retention&quot;
                                                </strong>{' '}
                                                prunes all report job history beyond the set
                                                retention limit, with the exception of these
                                                specific cases:
                                            </p>
                                            <ul
                                                className="pf-u-ml-md pf-u-mt-md"
                                                style={{ listStyleType: 'disclosure-closed ' }}
                                            >
                                                <li>
                                                    Jobs in the <strong>WAITING</strong> or{' '}
                                                    <strong>PREPARING</strong> state (unfinished
                                                    jobs)
                                                </li>
                                                <li>The last successful scheduled report job</li>
                                                <li>
                                                    The last successful on-demand emailed report job
                                                </li>
                                                <li>The last successful downloadable report job</li>
                                                <li>
                                                    Downloadable report jobs for which the report
                                                    file has not been deleted by either manual
                                                    deletion or by configuring the downloadable
                                                    report pruning settings
                                                </li>
                                            </ul>
                                        </div>
                                    }
                                    enableFlip
                                    position="top"
                                >
                                    <HelpIcon aria-label="Help for 'Vulnerability report job history retention' card" />
                                </Popover>
                            </FlexItem>
                        </Flex>
                    </CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={
                                privateConfig?.reportRetentionConfig?.historyRetentionDurationDays
                            }
                            suffix="day"
                            canRetainForever={false}
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat>
                    <CardTitle>
                        Prepared downloadable vulnerability reports retention days
                    </CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={
                                privateConfig?.reportRetentionConfig
                                    ?.downloadableReportRetentionDays
                            }
                            suffix="day"
                            canRetainForever={false}
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat>
                    <CardTitle>Prepared downloadable vulnerability reports limit</CardTitle>
                    <CardBody>
                        Set a total limit for all prepared downloadable vulnerability reports. Once
                        the limit is reached, the oldest report in download queue will be removed.
                    </CardBody>
                    <CardBody>
                        <DataRetentionValue
                            value={convertBetweenBytesAndMB(
                                privateConfig?.reportRetentionConfig
                                    ?.downloadableReportGlobalRetentionBytes,
                                'B'
                            )}
                            suffix="MB"
                            shouldPluralize={false}
                        />
                    </CardBody>
                </Card>
            </GridItem>
            {isFeatureFlagEnabled('ROX_ADMINISTRATION_EVENTS') && (
                <GridItem>
                    <Card isFlat>
                        <CardTitle>Administration events retention days</CardTitle>
                        <CardBody>
                            <DataRetentionValue
                                value={
                                    privateConfig?.administrationEventsConfig?.retentionDurationDays
                                }
                                suffix="day"
                                canRetainForever={false}
                            />
                        </CardBody>
                    </Card>
                </GridItem>
            )}
            <GridItem sm={12}>
                <Title headingLevel="h3" id="cluster-deletion">
                    Cluster deletion
                </Title>
            </GridItem>
            <GridItem>
                <Card isFlat>
                    <CardTitle>Decommissioned cluster age</CardTitle>
                    <CardBody>
                        <DataRetentionValue
                            value={
                                privateConfig?.decommissionedClusterRetention?.retentionDurationDays
                            }
                            suffix="day"
                        />
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem>
                <Card isFlat>
                    <CardTitle>Ignore clusters which have labels</CardTitle>
                    <CardBody>
                        {Object.keys(
                            privateConfig?.decommissionedClusterRetention?.ignoreClusterLabels ?? {}
                        ).length === 0 ? (
                            'No labels'
                        ) : (
                            <ClusterLabelsTable
                                labels={
                                    privateConfig.decommissionedClusterRetention.ignoreClusterLabels
                                }
                                hasAction={false}
                                handleChangeLabels={() => {}}
                            />
                        )}
                    </CardBody>
                    {isClustersRoutePathRendered && (
                        <CardBody>
                            <Button
                                variant="link"
                                isInline
                                component={LinkShim}
                                href={`${clustersBasePath}?s[Sensor Status]=UNHEALTHY`}
                            >
                                Clusters which have Sensor Status: Unhealthy
                            </Button>
                        </CardBody>
                    )}
                </Card>
            </GridItem>
        </Grid>
    );
};

export default PrivateConfigDataRetentionDetails;
