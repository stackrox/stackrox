import React, { ReactElement } from 'react';
import { Alert, Modal } from '@patternfly/react-core';

export type ManageStandardsErrorProp = {
    onClose: () => void;
    errorMessage: string;
};

function ManageStandardsError({ onClose, errorMessage }: ManageStandardsErrorProp): ReactElement {
    return (
        <Modal title="Manage standards" variant="small" isOpen onClose={onClose} showClose>
            <Alert title="Unable to fetch standards" variant="warning" isInline>
                {errorMessage}
            </Alert>
        </Modal>
    );
}

export default ManageStandardsError;
