import React, { useCallback } from 'react';
import { Table, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { Button, Divider, Flex, Form, Title } from '@patternfly/react-core';
import { useField } from 'formik';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import { integrationsPath } from 'routePaths';
import { fetchNotifierIntegrations } from 'services/NotifierIntegrationsService';
import { getTableUIState } from 'utils/getTableUIState';

function AttachNotifiersForm() {
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
        <Flex
            direction={{ default: 'column' }}
            spaceItems={{ default: 'spaceItemsNone' }}
            flexWrap={{ default: 'nowrap' }}
        >
            <Flex
                direction={{ default: 'column' }}
                spaceItems={{ default: 'spaceItemsSm' }}
                className="pf-v5-u-p-lg"
            >
                <Title headingLevel="h2">Notifiers</Title>
                <div>
                    Forward policy violations to external tooling by selecting one or more notifiers
                    from existing integrations.
                </div>
            </Flex>
            <Divider component="div" />
            <Form>
                <div className="pf-v5-u-p-lg">
                    <Table aria-label="Attach notifiers table" borders>
                        <Thead>
                            <Tr>
                                <Th
                                    select={{
                                        onSelect: onSelectAllNotifiers,
                                        isSelected:
                                            notifiers.length > 0 &&
                                            notifiers.length === field.value.length,
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
                                title: 'There was an error loading the collections',
                            }}
                            emptyProps={{
                                message:
                                    'No notifiers found. Add notifiers in the Integrations Page to add them to this policy.',
                                children: (
                                    <Button
                                        variant="secondary"
                                        component="a"
                                        target="_blank"
                                        href={integrationsPath}
                                    >
                                        Add a notifier
                                    </Button>
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
                                                        isSelected: field.value.includes(
                                                            notifier.id
                                                        ),
                                                    }}
                                                />
                                                <Td data-label="Notifier">{notifier.name}</Td>
                                                <Td data-label="Type">{notifier.type}</Td>
                                            </Tr>
                                        );
                                    })}
                                </Tbody>
                            )}
                        />
                    </Table>
                </div>
            </Form>
        </Flex>
    );
}

export default AttachNotifiersForm;
