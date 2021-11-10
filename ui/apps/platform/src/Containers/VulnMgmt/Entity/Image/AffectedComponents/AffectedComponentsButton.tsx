import React, { useState } from 'react';
import { Button } from '@patternfly/react-core';

import AffectedComponentsModal from './AffectedComponentsModal';

function AffectedComponentsButton({ components }) {
    const [isModalOpen, setIsModalOpen] = useState(false);

    function openModal() {
        setIsModalOpen(true);
    }

    function closeModal() {
        setIsModalOpen(false);
    }

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
