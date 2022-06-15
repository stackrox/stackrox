import React from 'react';
import { Link } from 'react-router-dom';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import { ListDeployment } from 'types/deployment.proto';
import { networkBasePath, riskBasePath } from 'routePaths';
import useURLSearch from 'hooks/useURLSearch';
import { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';

type DeploymentsAtMostRiskTableProps = {
    deployments: ListDeployment[];
};

const columnNames = {
    deployment: 'Deployment',
    resourceLocation: 'Resource location',
    riskPriority: 'Risk priority',
};

function linkToDeployment(id: string, name: string, searchFilter: SearchFilter): string {
    const query = getUrlQueryStringForSearchFilter({
        ...searchFilter,
        Deployment: name,
    });
    return `${riskBasePath}/${id}?${query}`;
}

function DeploymentsAtMostRiskTable({ deployments }: DeploymentsAtMostRiskTableProps) {
    const { searchFilter } = useURLSearch();
    return (
        <TableComposable
            aria-label="Deployments at most risk"
            variant="compact"
            borders={false}
            gridBreakPoint="grid-md"
        >
            <Thead>
                <Tr>
                    <Th className="pf-u-pl-0">{columnNames.deployment}</Th>
                    <Th>{columnNames.resourceLocation}</Th>
                    <Th className="pf-u-pr-0" textCenter>
                        {columnNames.riskPriority}
                    </Th>
                </Tr>
            </Thead>
            <Tbody>
                {deployments.map(({ id, name, cluster, namespace, priority }) => (
                    <Tr key={name}>
                        <Td className="pf-u-pl-0" dataLabel={columnNames.deployment}>
                            <Link to={linkToDeployment(id, name, searchFilter)}>{name}</Link>
                        </Td>
                        <Td dataLabel={columnNames.resourceLocation}>
                            <span>
                                in &ldquo;
                                <Link
                                    to={`${networkBasePath}/${id}`}
                                >{`${cluster} / ${namespace}`}</Link>
                                &rdquo;
                            </span>
                        </Td>
                        <Td className="pf-u-pr-0" textCenter dataLabel={columnNames.riskPriority}>
                            {priority}
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default DeploymentsAtMostRiskTable;
