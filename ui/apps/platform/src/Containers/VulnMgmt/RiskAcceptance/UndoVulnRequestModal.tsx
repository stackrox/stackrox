import React, { ReactElement, useState } from 'react';
import { Button, Modal, ModalVariant } from '@patternfly/react-core';

import FormMessage, { FormResponseMessage } from 'Components/PatternFly/FormMessage';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import pluralize from 'pluralize';

type AllowedType = 'DEFERRAL' | 'FALSE_POSITIVE';

export type UndoVulnRequestModalProps = {
    type: AllowedType;
    isOpen: boolean;
    numRequestsToBeAssessed: number;
    onSendRequest: () => Promise<FormResponseMessage>;
    onCompleteRequest: () => void;
    onCancel: () => void;
};

const typeLabel: Record<AllowedType, string> = {
    DEFERRAL: 'deferral',
    FALSE_POSITIVE: 'false positive',
};

function UndoVulnRequestModal({
    type,
    isOpen,
    numRequestsToBeAssessed,
    onSendRequest,
    onCompleteRequest,
    onCancel,
}: UndoVulnRequestModalProps): ReactElement {
    const [message, setMessage] = useState<FormResponseMessage>(null);

    function onHandleSubmit() {
        setMessage(null);
        onSendRequest()
            .then(() => {
                setMessage(null);
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

    const title = `Reobserve approved ${pluralize(typeLabel[type])} (${numRequestsToBeAssessed} )`;

    return (
        <Modal
            variant={ModalVariant.small}
            title={title}
            isOpen={isOpen}
            onClose={onCancelHandler}
            actions={[
                <Button key="confirm" variant="primary" onClick={onHandleSubmit}>
                    Reobserve CVE
                </Button>,
                <Button key="cancel" variant="link" onClick={onCancelHandler}>
                    Cancel
                </Button>,
            ]}
        >
            <FormMessage message={message} />
            <div>
                Reobserving an approved {typeLabel[type]} will return the CVE into the vulnerability
                management workflow
            </div>
        </Modal>
    );
}

export default UndoVulnRequestModal;
