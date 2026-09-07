import { useCallback } from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import { getVMCVEComponents } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import AdvisoryLinkOrText from '../../components/AdvisoryLinkOrText';
import FixedByVersion from '../../components/FixedByVersion';
import { sourceTypeLabels } from '../../constants';

export type AffectedComponentsTableProps = {
    virtualMachineId: string;
    cveId: string;
};

function AffectedComponentsTable({ virtualMachineId, cveId }: AffectedComponentsTableProps) {
    const colSpan = 5;
    const fetchComponents = useCallback(
        () => getVMCVEComponents(virtualMachineId, cveId),
        [virtualMachineId, cveId]
    );
    const { data, isLoading, error } = useRestQuery(fetchComponents);

    const tableState = getTableUIState({
        isLoading,
        data: data?.components,
        error,
        searchFilter: {},
    });

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>Component</Th>
                    <Th>Version</Th>
                    <Th>CVE fixed in</Th>
                    <Th>Advisory</Th>
                    <Th>Type</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                errorProps={{
                    title: 'There was an error loading affected components',
                }}
                emptyProps={{
                    message: 'There are no affected components',
                }}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map(
                            ({ componentName, componentVersion, source, fixedBy, advisory }) => (
                                <Tr key={`${componentName}-${componentVersion}`}>
                                    <Td dataLabel="Component">{componentName}</Td>
                                    <Td dataLabel="Version">{componentVersion}</Td>
                                    <Td dataLabel="CVE fixed in">
                                        <FixedByVersion fixedByVersion={fixedBy ?? ''} />
                                    </Td>
                                    <Td dataLabel="Advisory">
                                        <AdvisoryLinkOrText advisory={advisory} />
                                    </Td>
                                    <Td dataLabel="Type">{sourceTypeLabels[source]}</Td>
                                </Tr>
                            )
                        )}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default AffectedComponentsTable;
