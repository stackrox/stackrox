import type { ReactElement } from 'react';
import { getDateTime } from 'utils/dateUtils';

import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import type { IntegrationMergedItem } from '../utils/integrations';

type Props = {
    integrations: IntegrationMergedItem[];
};

const IntegrationsHealth = ({ integrations }: Props): ReactElement => {
    return (
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th width={20}>Name</Th>
                    <Th width={20}>Label</Th>
                    <Th width={45}>Error message</Th>
                    <Th width={15}>Date</Th>
                </Tr>
            </Thead>
            <Tbody data-testid="integration-healths">
                {integrations.map(({ id, name, label, errorMessage, lastTimestamp }) => (
                    <Tr key={id}>
                        <Td dataLabel="Name" modifier="breakWord" data-testid="integration-name">
                            {name}
                        </Td>
                        <Td dataLabel="Label" modifier="breakWord" data-testid="label">
                            {label}
                        </Td>
                        <Td dataLabel="Error message" modifier="breakWord">
                            {errorMessage.length === 0 ? '-' : errorMessage}
                        </Td>
                        <Td dataLabel="Date">{getDateTime(lastTimestamp)}</Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
};

export default IntegrationsHealth;
