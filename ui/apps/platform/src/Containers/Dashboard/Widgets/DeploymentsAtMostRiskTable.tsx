import React from 'react';
import { Link } from 'react-router-dom';
import { Truncate } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import { ListDeployment } from 'types/deployment.proto';
import { riskBasePath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { getURLLinkToDeployment } from 'Containers/NetworkGraph/utils/networkGraphURLUtils';

const columnNames = {
    deployment: 'Deployment',
    resourceLocation: 'Resource location',
    riskPriority: 'Risk priority',
};

function riskPageLinkToDeployment(id: string, name: string, searchFilter: SearchFilter): string {
    const query = getUrlQueryStringForSearchFilter({
        ...searchFilter,
        Deployment: name,
    });
    return `${riskBasePath}/${id}?${query}`;
}

type DeploymentsAtMostRiskTableProps = {
    deployments: ListDeployment[];
    searchFilter: SearchFilter;
};

function DeploymentsAtMostRiskTable({
    deployments,
    searchFilter,
}: DeploymentsAtMostRiskTableProps) {
    return (
        <TableComposable aria-label="Deployments at most risk" variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th className="pf-u-pl-0">{columnNames.deployment}</Th>
                    <Th>{columnNames.resourceLocation}</Th>
                    <Th className="pf-u-pr-0 pf-u-text-align-center-on-md">
                        {columnNames.riskPriority}
                    </Th>
                </Tr>
            </Thead>
            <Tbody>
                {deployments.map(({ id: deploymentId, name, cluster, namespace, priority }) => {
                    const networkGraphLink = getURLLinkToDeployment({
                        cluster,
                        namespace,
                        deploymentId,
                    });
                    return (
                        <Tr key={deploymentId}>
                            <Td className="pf-u-pl-0" dataLabel={columnNames.deployment}>
                                <Link
                                    to={riskPageLinkToDeployment(deploymentId, name, searchFilter)}
                                >
                                    <Truncate position="middle" content={name} />
                                </Link>
                            </Td>
                            <Td width={45} dataLabel={columnNames.resourceLocation}>
                                <span>
                                    in &ldquo;
                                    <Link to={networkGraphLink}>{`${cluster} / ${namespace}`}</Link>
                                    &rdquo;
                                </span>
                            </Td>
                            <Td
                                width={20}
                                className="pf-u-pr-0 pf-u-text-align-center-on-md"
                                dataLabel={columnNames.riskPriority}
                            >
                                {priority}
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </TableComposable>
    );
}

export default DeploymentsAtMostRiskTable;
