import React, { useState } from 'react';
import { Button } from '@patternfly/react-core';

import RequestCommentsModal from './RequestCommentsModal';

function RequestCommentsButton({ cve, comments }) {
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
                {comments.length} comments
            </Button>
            <RequestCommentsModal
                isOpen={isModalOpen}
                cve={cve}
                comments={comments}
                onClose={closeModal}
            />
        </>
    );
}

export default RequestCommentsButton;
