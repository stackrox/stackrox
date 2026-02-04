import { useState } from 'react';
import {
    Alert,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Text,
    Title,
} from '@patternfly/react-core';

import { deleteBaseImage as deleteBaseImageFn, getBaseImages } from 'services/BaseImagesService';
import type { BaseImageReference } from 'services/BaseImagesService';
import usePermissions from 'hooks/usePermissions';
import useAnalytics, {
    BASE_IMAGE_REFERENCE_ADD_MODAL_OPENED,
    BASE_IMAGE_REFERENCE_DELETED,
} from 'hooks/useAnalytics';
import useRestMutation from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ConfirmationModal from 'Components/PatternFly/ConfirmationModal/ConfirmationModal';
import PageTitle from 'Components/PageTitle';
import BaseImagesModal from './BaseImagesModal';
import BaseImagesTable from './BaseImagesTable';

/**
 * Page component for managing base images. Displays a list of approved base images
 * and provides functionality to add, edit, and delete base images.
 */
function BaseImagesPage() {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccess = hasReadWriteAccess('ImageAdministration');

    const [isAddModalOpen, setIsAddModalOpen] = useState(false);
    const [baseImageToEdit, setBaseImageToEdit] = useState<BaseImageReference | null>(null);
    const [baseImageToDelete, setBaseImageToDelete] = useState<BaseImageReference | null>(null);
    const { analyticsTrack } = useAnalytics();

    // Fetch base images on component mount
    const baseImagesRequest = useRestQuery(getBaseImages);

    // Delete mutation that refetches the list after successful deletion to keep UI in sync
    const deleteBaseImageMutation = useRestMutation((id: string) => deleteBaseImageFn(id), {
        onSuccess: () => {
            analyticsTrack(BASE_IMAGE_REFERENCE_DELETED);
            setBaseImageToDelete(null);
            baseImagesRequest.refetch();
        },
    });

    function onOpenAddModal() {
        analyticsTrack(BASE_IMAGE_REFERENCE_ADD_MODAL_OPENED);
        setIsAddModalOpen(true);
    }

    const onConfirmDelete = () => {
        if (baseImageToDelete) {
            deleteBaseImageMutation.mutate(baseImageToDelete.id);
        }
    };

    const onCancelDelete = () => {
        deleteBaseImageMutation.reset();
        setBaseImageToDelete(null);
    };

    const handleModalClose = () => {
        setIsAddModalOpen(false);
        setBaseImageToEdit(null);
    };

    const handleModalSuccess = () => {
        handleModalClose();
        baseImagesRequest.refetch();
    };

    const baseImages = baseImagesRequest.data ?? [];
    const isModalOpen = isAddModalOpen || baseImageToEdit !== null;

    return (
        <>
            <PageTitle title="Base Images" />
            <PageSection variant="light">
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    spaceItems={{ default: 'spaceItemsLg' }}
                >
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Base Images</Title>
                        <Text>
                            Manage approved base images for vulnerability tracking and
                            layer-specific filtering
                        </Text>
                    </FlexItem>
                    {hasWriteAccess && (
                        <Button variant="primary" onClick={onOpenAddModal}>
                            Add base image
                        </Button>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <BaseImagesTable
                    baseImages={baseImages}
                    hasWriteAccess={hasWriteAccess}
                    onEdit={setBaseImageToEdit}
                    onDelete={setBaseImageToDelete}
                    isActionInProgress={deleteBaseImageMutation.isLoading}
                    isLoading={baseImagesRequest.isLoading && !baseImagesRequest.data}
                    error={baseImagesRequest.error as Error | null}
                />
            </PageSection>
            <BaseImagesModal
                isOpen={isModalOpen}
                onClose={handleModalClose}
                onSuccess={handleModalSuccess}
                baseImageToEdit={baseImageToEdit}
            />
            {baseImageToDelete && (
                <ConfirmationModal
                    title="Delete base image?"
                    ariaLabel="Confirm delete base image"
                    confirmText="Delete"
                    isLoading={deleteBaseImageMutation.isLoading}
                    isOpen={baseImageToDelete !== null}
                    onConfirm={onConfirmDelete}
                    onCancel={onCancelDelete}
                >
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                    >
                        {deleteBaseImageMutation.isError && (
                            <Alert
                                variant="danger"
                                isInline
                                title="Error deleting base image"
                                component="p"
                            >
                                {getAxiosErrorMessage(deleteBaseImageMutation.error)}
                            </Alert>
                        )}
                        <p>
                            Permanently delete base image{' '}
                            <strong>
                                {baseImageToDelete.baseImageRepoPath}:
                                {baseImageToDelete.baseImageTagPattern}
                            </strong>
                            .
                        </p>
                    </Flex>
                </ConfirmationModal>
            )}
        </>
    );
}

export default BaseImagesPage;
