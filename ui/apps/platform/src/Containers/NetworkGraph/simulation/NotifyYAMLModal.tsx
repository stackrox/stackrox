import React from 'react';
import { Alert, Bullseye, Button, Modal, Spinner } from '@patternfly/react-core';

import { NetworkPolicyModification } from 'types/networkPolicy.proto';
import useTableSelection from 'hooks/useTableSelection';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { notifyNetworkPolicyModification } from 'services/NetworkService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import useFetchNotifiers from './useFetchNotifiers';

type NotifyYAMLModalProps = {
    isModalOpen: boolean;
    setIsModalOpen: React.Dispatch<React.SetStateAction<boolean>>;
    clusterId: string;
    modification: NetworkPolicyModification | null;
};

function NotifyYAMLModal({
    isModalOpen,
    setIsModalOpen,
    clusterId,
    modification,
}: NotifyYAMLModalProps): React.ReactElement {
    const { notifiers, isLoading, error } = useFetchNotifiers();
    const [errorMessage, setErrorMessage] = React.useState(error);
    const { selected, allRowsSelected, onSelect, onSelectAll, getSelectedIds, onClearAll } =
        useTableSelection(notifiers);

    function onNotify() {
        const notifierIds = getSelectedIds();
        notifyNetworkPolicyModification(clusterId, notifierIds, modification)
            .then(() => {
                onClose();
            })
            .catch((apiError) => {
                onClose();
                const message = getAxiosErrorMessage(apiError);
                const apiErrorMessage =
                    message || 'An unknown error occurred while sharing YAML with notifiers';
                setErrorMessage(apiErrorMessage);
            });
    }

    function onClose() {
        onClearAll();
        setErrorMessage(null);
        setIsModalOpen(!isModalOpen);
    }

    let content: React.ReactElement = <div />;

    if (isLoading) {
        content = (
            <Bullseye>
                <Spinner size="lg" />
            </Bullseye>
        );
    } else {
        content = (
            <Table aria-label="Notifiers table" variant="compact" borders>
                <Thead>
                    <Tr>
                        <Th
                            select={{
                                onSelect: onSelectAll,
                                isSelected: allRowsSelected,
                            }}
                        />
                        <Th>Name</Th>
                        <Th>Type</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {notifiers?.map((notifier, rowIndex) => {
                        return (
                            <Tr key={notifier.id}>
                                <Td
                                    select={{
                                        rowIndex,
                                        onSelect,
                                        isSelected: selected[rowIndex],
                                    }}
                                />
                                <Td dataLabel="Name">{notifier.name}</Td>
                                <Td dataLabel="Type">{notifier.type}</Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>
        );
    }

    return (
        <Modal
            variant="small"
            title="Share network policy YAML with team"
            isOpen={isModalOpen}
            onClose={onClose}
            actions={[
                <Button key="confirm" variant="primary" onClick={onNotify}>
                    Notify
                </Button>,
                <Button key="cancel" variant="link" onClick={onClose}>
                    Cancel
                </Button>,
            ]}
        >
            {errorMessage && (
                <Alert
                    isInline
                    variant="danger"
                    title={errorMessage}
                    component="p"
                    className="pf-v5-u-mb-lg"
                />
            )}
            {content}
        </Modal>
    );
}

export default NotifyYAMLModal;
