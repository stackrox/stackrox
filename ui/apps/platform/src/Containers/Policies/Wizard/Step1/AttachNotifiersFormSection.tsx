import React, { useEffect, useState } from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import { Title, Button, Bullseye } from '@patternfly/react-core';
import { useField } from 'formik';

import { integrationsPath } from 'routePaths';
import {
    fetchNotifierIntegrations,
    NotifierIntegrationBase,
} from 'services/NotifierIntegrationsService';

function AttachNotifiersFormSection() {
    const [notifiers, setNotifiers] = useState<NotifierIntegrationBase[]>([]);
    const [field, , helpers] = useField('notifiers');

    useEffect(() => {
        fetchNotifierIntegrations()
            .then(setNotifiers)
            .catch(() => {
                // TODO
            });

        return () => {
            setNotifiers([]);
        };
    }, []);

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
        <div className="pf-u-px-lg">
            <Title headingLevel="h2">Attach notifiers</Title>
            <div className="pf-u-mb-md pf-u-mt-sm">
                Forward policy violations to external tooling by selecting one or more notifiers
                from existing integrations.
            </div>
            {notifiers.length > 0 && (
                <TableComposable aria-label="Attach notifiers table" borders>
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAllNotifiers,
                                    isSelected: notifiers.length === field.value.length,
                                }}
                            />
                            <Th>Notifier</Th>
                            <Th>Type</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {notifiers.map((notifier, rowIndex) => (
                            <Tr key={notifier.id}>
                                <Td
                                    select={{
                                        rowIndex,
                                        onSelect: onSelectNotifier,
                                        isSelected: field.value.includes(notifier.id),
                                    }}
                                />
                                <Td data-label="Notifier">{notifier.name}</Td>
                                <Td data-label="Type">{notifier.type}</Td>
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            )}
            {notifiers.length === 0 && (
                <>
                    No notifiers found. Add notifiers in the Integrations Page to add them to this
                    policy.
                    <Bullseye className="pf-u-mt-md">
                        <Button
                            variant="secondary"
                            component="a"
                            target="_blank"
                            href={integrationsPath}
                        >
                            Add a notifier
                        </Button>
                    </Bullseye>
                </>
            )}
        </div>
    );
}

export default AttachNotifiersFormSection;
