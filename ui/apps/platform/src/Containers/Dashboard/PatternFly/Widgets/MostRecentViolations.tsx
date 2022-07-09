import React from 'react';
import { Link } from 'react-router-dom';
import { Flex, Title, Truncate } from '@patternfly/react-core';
import { TableComposable, Tbody, Tr, Td } from '@patternfly/react-table';
import { SecurityIcon } from '@patternfly/react-icons';

import ResourceIcon from 'Components/PatternFly/ResourceIcon';
import { DeploymentAlert } from 'types/alert.proto';
import { violationsBasePath } from 'routePaths';
import { severityColors } from 'constants/visuals/colors';
import { getDateTime } from 'utils/dateUtils';
import NoDataEmptyState from './NoDataEmptyState';

export type MostRecentViolationsProps = {
    alerts: DeploymentAlert[];
};

function MostRecentViolations({ alerts }: MostRecentViolationsProps) {
    return (
        <>
            <Title headingLevel="h5" className="pf-u-mb-sm">
                Most recent violations with critical severity
            </Title>
            {alerts.length > 0 ? (
                <TableComposable variant="compact" borders={false}>
                    <Tbody>
                        {alerts.map(({ id, time, deployment, policy }) => (
                            <Tr key={id}>
                                <Td className="pf-u-p-0" dataLabel="Severity icon">
                                    <SecurityIcon
                                        className="pf-u-display-inline"
                                        color={severityColors[policy.severity]}
                                    />
                                </Td>
                                <Td dataLabel="Violation name">
                                    <Link to={`${violationsBasePath}/${id}`}>
                                        <Truncate content={policy.name} />
                                    </Link>
                                </Td>
                                <Td dataLabel="Deployment in violation">
                                    <Flex
                                        direction={{ default: 'row' }}
                                        flexWrap={{ default: 'nowrap' }}
                                    >
                                        <ResourceIcon className="pf-u-mr-sm" kind="Deployment" />
                                        <Truncate content={deployment.name} />
                                    </Flex>
                                </Td>
                                <Td
                                    width={35}
                                    className="pf-u-pr-0 pf-u-text-align-right-on-md"
                                    dataLabel="Time of last violation occurrence"
                                >
                                    {getDateTime(time)}
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            ) : (
                <NoDataEmptyState />
            )}
        </>
    );
}

export default MostRecentViolations;
