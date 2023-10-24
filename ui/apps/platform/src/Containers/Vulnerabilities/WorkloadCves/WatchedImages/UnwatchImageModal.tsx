import React from 'react';
import { Alert, Button, Flex, Modal, Text } from '@patternfly/react-core';

import { unwatchImage } from 'services/imageService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

export type UnwatchImageModalProps = {
    unwatchImageName: string;
    isOpen: boolean;
    onClose: () => void;
    onWatchedImagesChange: () => void;
};

function UnwatchImageModal({
    unwatchImageName,
    isOpen,
    onClose,
    onWatchedImagesChange,
}: UnwatchImageModalProps) {
    const unwatchImageMutation = useRestMutation((name: string) => unwatchImage(name), {
        onSuccess: onWatchedImagesChange,
    });

    function onCloseModal() {
        onClose();
        unwatchImageMutation.reset();
    }

    return (
        <Modal
            title="Unwatch image"
            titleIconVariant="warning"
            isOpen={isOpen}
            onClose={onCloseModal}
            variant="small"
            showClose={false}
            actions={[
                <Button
                    key="Unwatch image"
                    isDisabled={unwatchImageMutation.isLoading || unwatchImageMutation.isSuccess}
                    isLoading={unwatchImageMutation.isLoading}
                    onClick={() => unwatchImageMutation.mutate(unwatchImageName)}
                >
                    Unwatch image
                </Button>,
                <Button variant="secondary" key="Close" onClick={onCloseModal}>
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                {unwatchImageMutation.isSuccess && (
                    <Alert
                        variant="success"
                        isInline
                        title="The image was successfully removed from the watch list"
                    />
                )}
                {unwatchImageMutation.isError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error removing the image from the watch list"
                    >
                        {getAxiosErrorMessage(unwatchImageMutation.error)}
                    </Alert>
                )}
                <Text>This will remove the following image from the watch list:</Text>
                <Text>{unwatchImageName}</Text>
            </Flex>
        </Modal>
    );
}

export default UnwatchImageModal;
