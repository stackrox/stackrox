import React from 'react';
import { Link } from 'react-router-dom';
import { Truncate } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import { ListDeployment } from 'types/deployment.proto';
import { networkBasePathPF, riskBasePath } from 'routePaths';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { getQueryString } from 'utils/queryStringUtils';

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
                {deployments.map(({ id, name, cluster, namespace, priority }) => {
                    // @TODO: Consider a more secure approach to creating links to the network graph so that
                    // areas outside of the Network Graph don't need to know the URL architecture of that feature
                    // Reference to discussion: https://github.com/stackrox/stackrox/pull/4955#discussion_r1112450278
                    const queryString = getQueryString({
                        s: {
                            Cluster: cluster,
                            Namespace: namespace,
                        },
                    });
                    const networkGraphLink = `${networkBasePathPF}/deployment/${id}${queryString}`;
                    return (
                        <Tr key={id}>
                            <Td className="pf-u-pl-0" dataLabel={columnNames.deployment}>
                                <Link to={riskPageLinkToDeployment(id, name, searchFilter)}>
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
