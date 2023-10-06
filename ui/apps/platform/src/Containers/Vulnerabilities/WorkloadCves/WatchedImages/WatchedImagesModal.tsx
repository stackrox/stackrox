import React, { CSSProperties, useCallback } from 'react';
import {
    Alert,
    Bullseye,
    Button,
    Divider,
    Flex,
    Modal,
    pluralize,
    Spinner,
    Text,
    Title,
} from '@patternfly/react-core';
import noop from 'lodash/noop';

import { getWatchedImages, unwatchImage, watchImage } from 'services/imageService';
import useRestMutation from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import WatchedImagesForm from './WatchedImagesForm';
import WatchedImagesTable from './WatchedImagesTable';

export type WatchedImagesModalProps = {
    defaultWatchedImageName: string;
    isOpen: boolean;
    onClose: () => void;
    onWatchedImagesChange: () => void;
};

function WatchedImagesModal({
    defaultWatchedImageName,
    isOpen,
    onClose,
    onWatchedImagesChange,
}: WatchedImagesModalProps) {
    const watchedImagesFn = useCallback(getWatchedImages, []);
    const currentWatchedImagesRequest = useRestQuery(watchedImagesFn);

    const watchImageMutation = useRestMutation((name: string) => watchImage(name), {
        onSuccess: onWatchedImagesChange,
    });

    const unwatchImageMutation = useRestMutation((name: string) => unwatchImage(name), {
        onSuccess: () => {
            onWatchedImagesChange();
            currentWatchedImagesRequest.refetch();
        },
    });

    function onCloseModal() {
        onClose();
        watchImageMutation.reset();
        unwatchImageMutation.reset();
    }

    const watchedImages = currentWatchedImagesRequest.data ?? [];

    return (
        <Modal
            aria-labelledby="manage-watched-images-modal-title"
            header={
                <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsXs' }}>
                    <Title id="manage-watched-images-modal-title" headingLevel="h2" size="2xl">
                        Manage watched images
                    </Title>
                    <Text>
                        Enter an image name to mark it as watched, so that it will continue to be
                        scanned even if no deployments use it.
                    </Text>
                </Flex>
            }
            isOpen={isOpen}
            onClose={onCloseModal}
            variant="medium"
            showClose={false}
            onEscapePress={watchImageMutation.isLoading ? noop : onCloseModal}
            actions={[
                <Button
                    key="Close"
                    onClick={onCloseModal}
                    isDisabled={watchImageMutation.isLoading}
                >
                    Close
                </Button>,
            ]}
        >
            <Flex direction={{ default: 'column' }}>
                {watchImageMutation.isSuccess && (
                    <Alert
                        variant="success"
                        isInline
                        title="The image was successfully added to the watch list"
                    >
                        {watchImageMutation.data ?? ''}
                    </Alert>
                )}
                {watchImageMutation.isError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error adding the image to the watch list"
                    >
                        {getAxiosErrorMessage(watchImageMutation.error)}
                    </Alert>
                )}
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
                {currentWatchedImagesRequest.error && (
                    <Alert
                        variant="danger"
                        isInline
                        title="There was an error loading the current list of watched images"
                    >
                        {getAxiosErrorMessage(currentWatchedImagesRequest.error)}
                    </Alert>
                )}
                <WatchedImagesForm
                    defaultWatchedImageName={defaultWatchedImageName}
                    watchImage={watchImageMutation.mutate}
                    watchedImagesRequest={currentWatchedImagesRequest}
                />
                <Divider component="div" />
                {currentWatchedImagesRequest.loading && !currentWatchedImagesRequest.data && (
                    <Bullseye>
                        <Spinner isSVG aria-label="Loading current watched images" />
                    </Bullseye>
                )}
                {currentWatchedImagesRequest.data && (
                    <>
                        <Title id="current-watched-images-list" headingLevel="h2">
                            {pluralize(watchedImages.length, 'watched image')}
                        </Title>
                        <WatchedImagesTable
                            aria-labelledby="current-watched-images-list"
                            className="pf-u-max-height"
                            style={
                                {
                                    overflowY: 'auto',
                                    '--pf-u-max-height--MaxHeight': '280px',
                                } as CSSProperties
                            }
                            watchedImages={watchedImages}
                            unwatchImage={unwatchImageMutation.mutate}
                            isUnwatchInProgress={unwatchImageMutation.isLoading}
                        />
                    </>
                )}
            </Flex>
        </Modal>
    );
}

export default WatchedImagesModal;
