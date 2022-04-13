import React from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';

import { enableDisableNotificationsForPolicies } from 'services/PoliciesService';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import { AlertVariantType } from 'hooks/patternfly/useToasts';
import useTableSelection from 'hooks/useTableSelection';
import { NotifierIntegration } from 'types/notifier.proto';

export type EnableDisableType = 'enable' | 'disable' | '';

type EnableDisableNotificationModalProps = {
    enableDisableType: EnableDisableType;
    setEnableDisableType: (type) => void;
    fetchPoliciesHandler: () => void;
    addToast: (text: string, variant: AlertVariantType, content?: string) => void;
    selectedPolicyIds: string[];
    notifiers: NotifierIntegration[];
};

function EnableDisableNotificationModal({
    enableDisableType,
    setEnableDisableType,
    fetchPoliciesHandler,
    addToast,
    selectedPolicyIds,
    notifiers,
}: EnableDisableNotificationModalProps) {
    // Handle selected rows in table
    const { selected, allRowsSelected, onSelect, onSelectAll, onClearAll, getSelectedIds } =
        useTableSelection(notifiers);

    const selectedNotifierIds = getSelectedIds();

    function enableDisableNotificationHandler() {
        const selectedNotifiers =
            notifiers.length === 1 ? notifiers.map((notifier) => notifier.id) : selectedNotifierIds;
        return enableDisableNotificationsForPolicies(
            selectedPolicyIds,
            selectedNotifiers,
            enableDisableType === 'disable'
        )
            .then(() => {
                fetchPoliciesHandler();
                addToast(`Successfully ${enableDisableType}d notification`, 'success');
            })
            .catch(({ response }) => {
                addToast(
                    `Could not ${enableDisableType} notification`,
                    'danger',
                    response.data.message
                );
            });
    }

    function onConfirmEnableDisableNotifications() {
        setEnableDisableType('');
        enableDisableNotificationHandler().finally(() => {});
    }

    function onCancelEnableDisableNotifications() {
        setEnableDisableType('');
        onClearAll();
    }

    return (
        <ConfirmationModal
            ariaLabel={`Confirm ${enableDisableType} notification`}
            confirmText={enableDisableType}
            title={`${enableDisableType} notifications`}
            isOpen={enableDisableType !== ''}
            isDestructiveAction={enableDisableType === 'disable'}
            onConfirm={onConfirmEnableDisableNotifications}
            onCancel={onCancelEnableDisableNotifications}
        >
            {notifiers.length === 1 ? (
                <>Selected notifier: {notifiers[0].name}</>
            ) : (
                <TableComposable
                    aria-label="Policies enable/disable notification table"
                    data-testid="policies-enable-disable-notification-table"
                >
                    <Thead>
                        <Tr>
                            <Th
                                select={{
                                    onSelect: onSelectAll,
                                    isSelected: allRowsSelected,
                                }}
                            />
                            <Th modifier="wrap">Select notifers</Th>
                            <Th />
                        </Tr>
                    </Thead>
                    <Tbody>
                        {notifiers.map(({ id, name }, rowIndex) => {
                            return (
                                <Tr key={id}>
                                    <Td
                                        select={{
                                            rowIndex,
                                            onSelect,
                                            isSelected: selected[rowIndex],
                                        }}
                                    />
                                    <Td>{name}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </TableComposable>
            )}
            {enableDisableType === 'disable' && (
                <div className="pf-u-pt-sm">
                    Are you sure you want to disable notification for {selectedPolicyIds.length}{' '}
                    {pluralize('policy', selectedPolicyIds.length)}?
                </div>
            )}
        </ConfirmationModal>
    );
}

export default EnableDisableNotificationModal;
