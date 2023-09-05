import React from 'react';
import { Button, Modal } from '@patternfly/react-core';

export type WatchedImagesModalProps = {
    defaultWatchedImageName: string;
    isOpen: boolean;
    onClose: () => void;
};

function WatchedImagesModal({ defaultWatchedImageName, isOpen, onClose }: WatchedImagesModalProps) {
    return (
        <Modal
            title="Manage watched images"
            isOpen={isOpen}
            onClose={onClose}
            variant="medium"
            actions={[
                <Button key="Close" onClick={onClose}>
                    Close
                </Button>,
            ]}
        >
            <div>Image name: {defaultWatchedImageName}</div>
        </Modal>
    );
}

export default WatchedImagesModal;
