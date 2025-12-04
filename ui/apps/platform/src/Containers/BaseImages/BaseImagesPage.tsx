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

import {
    addBaseImage as addBaseImageFn,
    deleteBaseImage as deleteBaseImageFn,
    getBaseImages,
} from 'services/BaseImagesService';
import useRestMutation from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import BaseImagesModal from './BaseImagesModal';
import BaseImagesTable from './BaseImagesTable';

function BaseImagesPage() {
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);

    const baseImagesRequest = useRestQuery(useCallback(getBaseImages, []));

    const addBaseImageMutation = useRestMutation(
        (data: { baseImageRepoPath: string; baseImageTagPattern: string }) =>
            addBaseImageFn(data.baseImageRepoPath, data.baseImageTagPattern),
        {
            onSuccess: () => {
                baseImagesRequest.refetch();
            },
        }
    );

    const deleteBaseImageMutation = useRestMutation((id: string) => deleteBaseImageFn(id), {
        onSuccess: () => {
            baseImagesRequest.refetch();
        },
    });

    const handleAddBaseImageSuccess = () => {
        setIsAddModalOpen(false);
        baseImagesRequest.refetch();
    };

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
                    <Button variant="primary" onClick={handleOpenAddModal}>
                        Add base image
                    </Button>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                {deleteBaseImageMutation.isError && (
                    <Alert
                        variant="danger"
                        isInline
                        title="Error removing base image"
                        component="p"
                        className="pf-v5-u-mb-lg"
                    >
                        {getAxiosErrorMessage(deleteBaseImageMutation.error)}
                    </Alert>
                )}
                <BaseImagesTable
                    baseImages={baseImages}
                    onRemove={(baseImage) => handleRemoveBaseImage(baseImage.id)}
                    isRemoveInProgress={deleteBaseImageMutation.isLoading}
                    isLoading={baseImagesRequest.isLoading && !baseImagesRequest.data}
                    error={baseImagesRequest.error as Error | null}
                />
            </PageSection>
            <BaseImagesModal
                isOpen={isAddModalOpen}
                onClose={handleCloseAddModal}
                onSuccess={handleAddBaseImageSuccess}
            />
        </>
    );
}

export default BaseImagesPage;
