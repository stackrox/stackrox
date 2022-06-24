import React from 'react';
import { Link } from 'react-router-dom';
import {
    EmptyState,
    EmptyStateIcon,
    EmptyStateVariant,
    Flex,
    Title,
    Truncate,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Tr, Td } from '@patternfly/react-table';

import { DeploymentAlert } from 'types/alert.proto';
import { violationsBasePath } from 'routePaths';
import { SearchIcon, SecurityIcon } from '@patternfly/react-icons';
import { severityColors } from 'constants/visuals/colors';
import { getDateTime } from 'utils/dateUtils';

export type MostRecentViolationsProps = {
    alerts: Partial<DeploymentAlert>[];
};

function MostRecentViolations({ alerts }: MostRecentViolationsProps) {
    if (alerts.length === 0) {
        return (
            <EmptyState variant={EmptyStateVariant.xs}>
                <EmptyStateIcon icon={SearchIcon} />
                <Title headingLevel="h4" size="md">
                    No critical violations were found in the selected scope
                </Title>
            </EmptyState>
        );
    }

    return (
        <>
            <Title headingLevel="h5" className="pf-u-mb-sm">
                Most recent violations with critical severity
            </Title>
            <TableComposable variant="compact" borders={false}>
                <Tbody>
                    {alerts.map(({ id, time, deployment, policy }) => (
                        <Tr key={id}>
                            <Td className="pf-u-p-0" dataLabel="Severity icon">
                                {policy && (
                                    <SecurityIcon
                                        className="pf-u-display-inline"
                                        color={severityColors[policy.severity]}
                                    />
                                )}
                            </Td>
                            <Td dataLabel="Violation name">
                                {policy && (
                                    <Flex direction={{ default: 'row' }}>
                                        <Link to={`${violationsBasePath}`}>
                                            <Truncate content={policy.name} />
                                        </Link>
                                    </Flex>
                                )}
                            </Td>
                            <Td dataLabel="Deployment in violation">
                                <Truncate content={deployment?.name ?? ''} />
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
        </>
    );
}

export default MostRecentViolations;
