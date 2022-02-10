import React, { ReactElement, useState } from 'react';
import { Button, Modal, ModalVariant } from '@patternfly/react-core';
import pluralize from 'pluralize';

import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type AllowedType = 'DEFERRAL' | 'FALSE_POSITIVE';

export type CancelVulnRequestModalProps = {
    type: AllowedType;
    numRequestsToBeAssessed: number;
    onSendRequest: () => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

const typeLabel: Record<AllowedType, string> = {
    DEFERRAL: 'deferral',
    FALSE_POSITIVE: 'false positive',
};

function CancelVulnRequestModal({
    type,
    numRequestsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: CancelVulnRequestModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);

    function onHandleSubmit() {
        setMessage(null);
        onSendRequest()
            .then(() => {
                onCompleteRequest();
            })
            .catch((response) => {
                const error = new Error(response.message);
                setMessage({
                    message: getAxiosErrorMessage(error),
                    isError: true,
                });
            });
    }

    function onCancelHandler() {
        setMessage(null);
        onCancel();
    }

    const title = `Cancel ${pluralize(
        typeLabel[type],
        numRequestsToBeAssessed
    )} (${numRequestsToBeAssessed})`;

    return (
        <Modal
            variant={ModalVariant.small}
            title={title}
            isOpen
            onClose={onCancelHandler}
            actions={[
                <Button key="confirm" variant="primary" onClick={onHandleSubmit}>
                    Submit cancellation
                </Button>,
                <Button key="cancel" variant="link" onClick={onCancelHandler}>
                    Cancel
                </Button>,
            ]}
        >
            <FormMessage message={message} />
            <div>
                Canceling a pending {typeLabel[type]} request will return the CVE into the
                vulnerability management workflow
            </div>
        </Modal>
    );
}

export default CancelVulnRequestModal;
