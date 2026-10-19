import type { ReactElement } from 'react';
import { Alert, Button, Flex } from '@patternfly/react-core';
import { Modal } from '@patternfly/react-core/deprecated';

import SecureClusterUsingOperator from './SecureClusterUsingOperator';

const headingLevel = 'h2';

export type SecureClusterModalProps = {
    isModalOpen: boolean;
    setIsModalOpen: (isOpen: boolean) => void;
};

function SecureClusterModal({ isModalOpen, setIsModalOpen }): ReactElement {
    function onClose() {
        setIsModalOpen(false);
    }

    return (
        <Modal
            variant="medium"
            title="View installation methods"
            isOpen={isModalOpen}
            onClose={onClose}
            actions={[
                <Button key="Close" variant="primary" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                <SecureClusterUsingOperator headingLevel={headingLevel} />
                <Alert
                    variant="info"
                    isInline
                    title="You can use a single cluster registration secret to register multiple clusters."
                    component="p"
                />
            </Flex>
        </Modal>
    );
}

export default SecureClusterModal;
