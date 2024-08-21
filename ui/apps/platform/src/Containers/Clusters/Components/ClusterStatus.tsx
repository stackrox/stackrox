import React, { ReactElement } from 'react';
import { Button, Popover, PopoverPosition } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons/dist/esm/icons';

import ExternalLink from 'Components/PatternFly/IconText/ExternalLink';
import PopoverBodyContent from 'Components/PopoverBodyContent';
import useMetadata from 'hooks/useMetadata';
import { healthStatusLabels } from 'messages/common';
import { getVersionedDocs } from 'utils/versioning';

import HealthStatus from './HealthStatus';
import ClusterStatusPill from './ClusterStatusPill';
import { healthStatusStyles } from '../cluster.helpers';
import { ClusterHealthStatus } from '../clusterTypes';

/*
 * Cluster Status in Clusters list or Cluster side panel
 *
 * Caller is responsible for optional chaining in case healthStatus is null.
 */

type ClusterStatusProps = {
    healthStatus: ClusterHealthStatus;
    isList?: boolean;
};

function ClusterStatus({ healthStatus, isList = false }: ClusterStatusProps): ReactElement {
    const { version } = useMetadata();

    const { overallHealthStatus = 'UNAVAILABLE' } = healthStatus ?? {};

    const { Icon, fgColor } = healthStatusStyles[overallHealthStatus];
    const icon = <Icon className={`${isList ? 'inline' : ''} h-4 w-4`} />;

    const unhealthyClusterDetailAvailable = overallHealthStatus === 'UNHEALTHY';
    const bodyContent = version ? (
        <ExternalLink>
            <a
                href={getVersionedDocs(
                    version,
                    'troubleshooting/retrieving-and-analyzing-the-collector-logs-and-pod-status.html'
                )}
                target="_blank"
                rel="noopener noreferrer"
            >
                Troubleshooting collector
            </a>
        </ExternalLink>
    ) : (
        <span>Documentation not available; version missing</span>
    );

    return (
        <div>
            <div className={`${isList ? 'mb-1' : ''}`}>
                <HealthStatus icon={icon} iconColor={fgColor} isList={isList}>
                    <div data-testid="clusterStatus" className={`${isList ? 'inline' : ''}`}>
                        <span>
                            {unhealthyClusterDetailAvailable ? (
                                <Popover
                                    aria-label="Unhealthy Collector, with link to troubleshooting"
                                    className="widget-options-menu"
                                    minWidth="0px"
                                    position={PopoverPosition.top}
                                    enableFlip
                                    bodyContent={
                                        <PopoverBodyContent
                                            headerContent={
                                                <span className="pf-v5-u-danger-color-100">
                                                    Unhealthy Collector
                                                </span>
                                            }
                                            headerIcon={
                                                <ExclamationCircleIcon className="pf-v5-u-danger-color-100" />
                                            }
                                            bodyContent={bodyContent}
                                        />
                                    }
                                >
                                    <Button
                                        aria-label="Show troubleshooting info"
                                        variant="link"
                                        className="pf-v5-u-mr-sm"
                                        isInline
                                    >
                                        <span>{healthStatusLabels[overallHealthStatus]}</span>
                                    </Button>
                                </Popover>
                            ) : (
                                <span>{healthStatusLabels[overallHealthStatus]}</span>
                            )}
                        </span>
                    </div>
                </HealthStatus>
            </div>
            {isList && <ClusterStatusPill healthStatus={healthStatus} />}
        </div>
    );
}

export default ClusterStatus;
