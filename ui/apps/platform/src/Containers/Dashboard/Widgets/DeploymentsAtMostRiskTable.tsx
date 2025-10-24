import { Link } from 'react-router-dom-v5-compat';
import { Truncate } from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ListDeployment } from 'types/deployment.proto';
import { riskBasePath } from 'routePaths';
import type { SearchFilter } from 'types/search';
import { getUrlQueryStringForSearchFilter } from 'utils/searchUtils';
import { getURLLinkToDeployment } from 'Containers/NetworkGraph/utils/networkGraphURLUtils';

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
        <Table aria-label="Deployments at most risk" variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th className="pf-v5-u-pl-0">Deployment</Th>
                    <Th>Resource location</Th>
                    <Th className="pf-v5-u-pr-0 pf-v5-u-text-align-center-on-md">Risk priority</Th>
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
                            <Td className="pf-v5-u-pl-0" dataLabel="Deployment">
                                <Link
                                    to={riskPageLinkToDeployment(deploymentId, name, searchFilter)}
                                >
                                    <Truncate position="middle" content={name} />
                                </Link>
                            </Td>
                            <Td width={45} dataLabel="Resource location">
                                <span>
                                    in &ldquo;
                                    <Link to={networkGraphLink}>{`${cluster} / ${namespace}`}</Link>
                                    &rdquo;
                                </span>
                            </Td>
                            <Td
                                width={20}
                                className="pf-v5-u-pr-0 pf-v5-u-text-align-center-on-md"
                                dataLabel="Risk priority"
                            >
                                {priority}
                            </Td>
                        </Tr>
                    );
                })}
            </Tbody>
        </Table>
    );
}

export default DeploymentsAtMostRiskTable;
