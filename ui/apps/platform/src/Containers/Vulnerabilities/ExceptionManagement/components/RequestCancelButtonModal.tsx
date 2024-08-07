import React, { useState } from 'react';
import { Alert, AlertVariant, Button, Flex, Modal, Text } from '@patternfly/react-core';

import {
    VulnerabilityException,
    cancelVulnerabilityException,
} from 'services/VulnerabilityExceptionService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useModal from 'hooks/useModal';
import useRestMutation from 'hooks/useRestMutation';

type RequestCancelButtonModalProps = {
    exception: VulnerabilityException;
    onSuccess: (vulnerabilityException: VulnerabilityException) => void;
};

function RequestCancelButtonModal({ exception, onSuccess }: RequestCancelButtonModalProps) {
    const cancelRequestMutation = useRestMutation(cancelVulnerabilityException);

    const { isModalOpen, openModal, closeModal } = useModal();
    const [errorMessage, setErrorMessage] = useState<string | null>(null);

    function cancelRequest() {
        const payload = {
            id: exception.id,
        };
        cancelRequestMutation.mutate(payload, {
            onSuccess: (exception) => {
                onSuccess(exception);
                closeModal();
            },
            onError: (error) => {
                setErrorMessage(getAxiosErrorMessage(error));
            },
        });
    }

    return (
        <>
            <Button variant="secondary" onClick={openModal}>
                Cancel request
            </Button>
            <Modal
                variant="small"
                title="Cancel request"
                isOpen={isModalOpen}
                onClose={closeModal}
                actions={[
                    <Button
                        key="approve"
                        variant="primary"
                        isLoading={cancelRequestMutation.isLoading}
                        isDisabled={cancelRequestMutation.isLoading}
                        onClick={cancelRequest}
                    >
                        Cancel request
                    </Button>,
                    <Button key="cancel" variant="link" onClick={closeModal}>
                        Cancel
                    </Button>,
                ]}
                showClose={false}
            >
                <Flex className="pf-v5-u-py-md">
                    {errorMessage && (
                        <Alert
                            isInline
                            variant={AlertVariant.danger}
                            title={errorMessage}
                            component="p"
                        />
                    )}
                    <Alert
                        variant="warning"
                        isInline
                        title="Cancelling the request will return the CVEs to the 'Observed' status."
                        component="p"
                    >
                        <Text>CVE count: {exception.cves.length}</Text>
                    </Alert>
                </Flex>
            </Modal>
        </>
    );
}

export default RequestCancelButtonModal;
