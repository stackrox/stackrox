import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import type { ProcessIndicator } from 'types/processIndicator.proto';
import { getDateTime } from 'utils/dateUtils';

export type SignalProps = {
    signals: ProcessIndicator[];
};

function Signal({ signals }: SignalProps) {
    return (
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th>Time</Th>
                    <Th>Pod ID</Th>
                    <Th>UID</Th>
                </Tr>
            </Thead>
            <Tbody>
                {signals.map((processSignal) => (
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
        </Table>
    );
}

export default Signal;
