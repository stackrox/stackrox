import React, { useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Table, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { Form } from '@patternfly/react-core';
import { useField } from 'formik';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import { integrationsPath } from 'routePaths';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { getTableUIState } from 'utils/getTableUIState';

function NotifiersForm() {
    const [field, , helpers] = useField('notifiers');

    const fetchNotifiers = useCallback(() => fetchNotifierIntegrations(), []);
    const { data: notifiers = [], isLoading, error } = useRestQuery(fetchNotifiers);

    const tableState = getTableUIState({
        isLoading,
        data: notifiers,
        error,
        searchFilter: {},
    });

    function onSelectNotifier(e, isSelected, rowIndex) {
        const selectedNotifiers = field.value || [];
        if (isSelected) {
            helpers.setValue([...selectedNotifiers, notifiers[rowIndex].id]);
        } else {
            helpers.setValue(selectedNotifiers.filter((id) => id !== notifiers[rowIndex].id));
        }
    }

    function onSelectAllNotifiers(e, isSelected) {
        if (isSelected) {
            helpers.setValue(notifiers.map((notifier) => notifier.id));
        } else {
            helpers.setValue([]);
        }
    }

    return (
        <Form>
            <Table borders>
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                onSelect: onSelectAllNotifiers,
                                isSelected:
                                    notifiers.length > 0 && notifiers.length === field.value.length,
                            }}
                            modifier="nowrap"
                        />
                        <Th>Notifier</Th>
                        <Th>Type</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={3}
                    errorProps={{
                        title: 'There was an error loading notifiers',
                    }}
                    emptyProps={{
                        message:
                            'No notifiers found. Add notifiers in the Integrations Page to add them to this policy.',
                        children: (
                            <Link to={integrationsPath} target="_blank" rel="noopener noreferrer">
                                Go to integrations
                            </Link>
                        ),
                    }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((notifier, rowIndex) => {
                                return (
                                    <Tr key={notifier.id}>
                                        <Td
                                            select={{
                                                rowIndex,
                                                onSelect: onSelectNotifier,
                                                isSelected: field.value.includes(notifier.id),
                                            }}
                                        />
                                        <Td dataLabel="Notifier">{notifier.name}</Td>
                                        <Td dataLabel="Type">{notifier.type}</Td>
                                    </Tr>
                                );
                            })}
                        </Tbody>
                    )}
                />
            </Table>
        </Form>
    );
}

export default NotifiersForm;
