import React from 'react';
import { Alert, Button, Flex, Modal } from '@patternfly/react-core';
import noop from 'lodash/noop';

import { watchImage } from 'services/imageService';
import useRestMutation from 'hooks/useRestMutation';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import WatchedImagesForm from './WatchedImagesForm';

export type WatchedImagesModalProps = {
    defaultWatchedImageName: string;
    isOpen: boolean;
    onClose: () => void;
};

function WatchedImagesModal({ defaultWatchedImageName, isOpen, onClose }: WatchedImagesModalProps) {
    const { data, mutate, isSuccess, isLoading, isError, error, reset } = useRestMutation(
        (name: string) => watchImage(name)
    );

    function onCloseModal() {
        onClose();
        reset();
    }

    return (
        <Modal
            title="Manage watched images"
            isOpen={isOpen}
            onClose={onCloseModal}
            variant="medium"
            showClose={!isLoading}
            onEscapePress={isLoading ? noop : onCloseModal}
            actions={[
                <Button key="Close" onClick={onCloseModal} isDisabled={isLoading}>
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                {isSuccess && (
                    <Alert
                        variant="success"
                        isInline
                        title="The image was successfully added to the watch list"
                    >
                        {data ?? ''}
                    </Alert>
                )}
                {isError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error adding the image to the watch list"
                    >
                        {getAxiosErrorMessage(error)}
                    </Alert>
                )}
                <WatchedImagesForm
                    defaultWatchedImageName={defaultWatchedImageName}
                    watchImage={mutate}
                />
            </Flex>
        </Modal>
    );
}

export default WatchedImagesModal;
