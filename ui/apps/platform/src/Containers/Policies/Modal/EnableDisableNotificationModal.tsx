import React from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';
import capitalize from 'lodash/capitalize';

import { enableDisableNotificationsForPolicies } from 'services/PoliciesService';
import ConfirmationModal from 'Components/PatternFly/ConfirmationModal';
import { AlertVariantType } from 'hooks/patternfly/useToasts';
import useTableSelection from 'hooks/useTableSelection';
import { NotifierIntegration } from 'types/notifier.proto';

export type EnableDisableType = 'enable' | 'disable';

type EnableDisableNotificationModalProps = {
    enableDisableType: EnableDisableType | null;
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

    const selectedNotifierIds = notifiers.length === 1 ? [notifiers[0].id] : getSelectedIds();
    const enableDisableTypeText = enableDisableType as string;

    function enableDisableNotificationHandler() {
        return enableDisableNotificationsForPolicies(
            selectedPolicyIds,
            selectedNotifierIds,
            enableDisableType === 'disable'
        )
            .then(() => {
                fetchPoliciesHandler();
                addToast(`Successfully ${enableDisableTypeText}d notification`, 'success');
            })
            .catch(({ response }) => {
                addToast(
                    `Could not ${enableDisableTypeText} notification`,
                    'danger',
                    response.data.message
                );
            });
    }

    function onConfirmEnableDisableNotifications() {
        setEnableDisableType(null);
        enableDisableNotificationHandler().finally(() => {});
    }

    function onCancelEnableDisableNotifications() {
        setEnableDisableType(null);
        onClearAll();
    }

    return (
        <ConfirmationModal
            ariaLabel={`Confirm ${enableDisableTypeText} notification`}
            confirmText={enableDisableTypeText}
            title={`${capitalize(enableDisableTypeText)} notification`}
            isOpen={enableDisableType !== null}
            isDestructive={enableDisableType === 'disable'}
            onConfirm={onConfirmEnableDisableNotifications}
            onCancel={onCancelEnableDisableNotifications}
            isConfirmDisabled={
                notifiers.length === 0 || (notifiers.length > 0 && selectedNotifierIds.length === 0)
            }
        >
            {notifiers.length === 0 && <>No notifiers configured!</>}
            {notifiers.length === 1 && <>Selected notifier: {notifiers[0].name}</>}
            {notifiers.length > 1 && (
                <TableComposable
                    aria-label="Policies enable/disable notification table"
                    data-testid="policies-enable-disable-notification-table"
                    variant="compact"
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
            {notifiers.length > 0 && enableDisableType === 'disable' && (
                <div className="pf-u-pt-sm">
                    Are you sure you want to disable notification for {selectedPolicyIds.length}{' '}
                    {pluralize('policy', selectedPolicyIds.length)}?
                </div>
            )}
        </ConfirmationModal>
    );
}

export default EnableDisableNotificationModal;
