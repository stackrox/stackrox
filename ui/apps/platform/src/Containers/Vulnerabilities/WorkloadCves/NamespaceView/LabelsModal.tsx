import React from 'react';
import { Button, List, ListItem, Modal, pluralize } from '@patternfly/react-core';
import useModal from 'hooks/useModal';

export type DeploymentFilterLinkProps = {
    labels: {
        key: string;
        value: string;
    }[];
};

function LabelsModal({ labels }) {
    const { isModalOpen, openModal, closeModal } = useModal();

    const text = pluralize(labels.length, 'label');

    return (
        <>
            <Button variant="link" isInline onClick={openModal}>
                {text}
            </Button>
            <Modal
                variant="small"
                title={text}
                isOpen={isModalOpen}
                onClose={closeModal}
                actions={[
                    <Button key="cancel" variant="primary" onClick={closeModal}>
                        Cancel
                    </Button>,
                ]}
            >
                <List isPlain isBordered className="pf-u-py-sm">
                    {labels.map((label) => {
                        const labelText = `${label.key}: ${label.value}`;
                        return <ListItem key={labelText}>{labelText}</ListItem>;
                    })}
                </List>
            </Modal>
        </>
    );
}

export default LabelsModal;
