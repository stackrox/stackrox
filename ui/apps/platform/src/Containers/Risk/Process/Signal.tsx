import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { ProcessIndicator } from 'types/processIndicator.proto';
import { getDateTime } from 'utils/dateUtils';
import { getTableUIState } from 'utils/getTableUIState';

export type SignalProps = {
    signals: ProcessIndicator[];
};

function Signal({ signals }: SignalProps) {
    const tableState = getTableUIState({
        isLoading: false,
        data: signals,
        error: undefined,
        searchFilter: {},
    });

    return (
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th>Time</Th>
                    <Th>Pod ID</Th>
                    <Th>UID</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={3}
                // signals is always non-empty, so this empty state should not render.
                // The backend creates a ProcessGroup only from one or more indicators.
                emptyProps={{ title: 'No process activity recorded', message: '' }}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((processSignal) => (
                            <Tr key={processSignal.id}>
                                <Td dataLabel="Time">
                                    {processSignal.signal.time
                                        ? getDateTime(processSignal.signal.time)
                                        : '-'}
                                </Td>
                                <Td dataLabel="Pod ID">{processSignal.podId}</Td>
                                <Td dataLabel="UID">{processSignal.signal.uid}</Td>
                            </Tr>
                        ))}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default Signal;
