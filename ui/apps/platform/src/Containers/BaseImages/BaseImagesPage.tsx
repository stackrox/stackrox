import { useCallback, useState } from 'react';
import type { AxiosError } from 'axios';
import {
    Alert,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Spinner,
    Text,
    Title,
} from '@patternfly/react-core';

import {
    addBaseImage as addBaseImageFn,
    deleteBaseImage as deleteBaseImageFn,
    getBaseImages,
} from 'services/BaseImagesService';
import type { AddBaseImageRequest } from 'services/BaseImagesService';
import useRestMutation from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import BaseImagesModal from './BaseImagesModal';
import BaseImagesTable from './BaseImagesTable';

function BaseImagesPage() {
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);

    // Fetch base images list
    const baseImagesRequest = useRestQuery(useCallback(getBaseImages, []));

    // Add base image mutation
    const addBaseImageMutation = useRestMutation(
        (request: AddBaseImageRequest) => addBaseImageFn(request),
        {
            onSuccess: () => {
                baseImagesRequest.refetch();
            },
        }
    );

    // Delete base image mutation
    const deleteBaseImageMutation = useRestMutation((id: string) => deleteBaseImageFn(id), {
        onSuccess: () => {
            baseImagesRequest.refetch();
        },
    });

    function handleCloseAddModal() {
        setIsAddModalOpen(false);
    }

    function handleOpenAddModal() {
        addBaseImageMutation.reset();
        setIsAddModalOpen(true);
    }

    function handleRemoveBaseImage(id: string) {
        deleteBaseImageMutation.mutate(id);
    }

    const baseImages = baseImagesRequest.data ?? [];

    return (
        <>
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
                    <FlexItem align={{ default: 'alignRight' }}>
                        <Button variant="primary" onClick={handleOpenAddModal}>
                            Add base image
                        </Button>
                    </FlexItem>
                </Flex>
            </PageSection>

            <Divider component="div" />

            <PageSection className="pf-v5-u-display-flex pf-v5-u-flex-direction-column">
                <Flex
                    direction={{ default: 'column' }}
                    spaceItems={{ default: 'spaceItemsLg' }}
                    className="pf-v5-u-flex-grow-1"
                >
                    {/* Alerts */}
                    {deleteBaseImageMutation.isError && (
                        <Alert
                            variant="danger"
                            isInline
                            title="Error removing base image"
                            component="p"
                        >
                            {getAxiosErrorMessage(deleteBaseImageMutation.error)}
                        </Alert>
                    )}

                    {/* Loading spinner */}
                    {baseImagesRequest.isLoading && !baseImagesRequest.data && (
                        <Bullseye>
                            <Spinner aria-label="Loading base images" />
                        </Bullseye>
                    )}

                    {/* Table or Empty State */}
                    {baseImagesRequest.data !== undefined && (
                        <BaseImagesTable
                            baseImages={baseImages}
                            onRemove={(baseImage) => handleRemoveBaseImage(baseImage.id)}
                            isRemoveInProgress={deleteBaseImageMutation.isLoading}
                        />
                    )}
                </Flex>
            </PageSection>

            {/* Add Modal */}
            <BaseImagesModal
                isOpen={isAddModalOpen}
                onClose={handleCloseAddModal}
                isSuccess={addBaseImageMutation.isSuccess}
                isError={addBaseImageMutation.isError}
                error={(addBaseImageMutation.error as AxiosError) || null}
                formProps={{
                    onAddBaseImage: addBaseImageMutation.mutate,
                    isSubmitting: addBaseImageMutation.isLoading,
                    error: (addBaseImageMutation.error as AxiosError) || null,
                }}
            />
        </>
    );
}

export default BaseImagesPage;
