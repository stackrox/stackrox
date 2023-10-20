import React from 'react';
import { Button, Modal } from '@patternfly/react-core';
import { BaseVulnerabilityException } from 'services/VulnerabilityExceptionService';

export type CompletedExceptionRequestModalProps = {
    exceptionRequest: BaseVulnerabilityException;
    onClose: () => void;
};

function CompletedExceptionRequestModal({
    exceptionRequest,
    onClose,
}: CompletedExceptionRequestModalProps) {
    const title = 'TODO';

    return (
        <Modal
            onClose={onClose}
            title={title}
            isOpen
            variant="medium"
            actions={[
                <Button key="confirm" variant="primary" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            {exceptionRequest.id}
        </Modal>
    );
}

export default CompletedExceptionRequestModal;
