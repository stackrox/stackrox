import React from 'react';
import { Button } from '@patternfly/react-core';

import useModal from 'hooks/useModal';
import AffectedComponentsModal from './AffectedComponentsModal';

function AffectedComponentsButton({ components }) {
    const { isModalOpen, openModal, closeModal } = useModal();

    return (
        <>
            <Button variant="link" isInline onClick={openModal}>
                {components.length} components
            </Button>
            <AffectedComponentsModal
                isOpen={isModalOpen}
                components={components}
                onClose={closeModal}
            />
        </>
    );
}

export default AffectedComponentsButton;
