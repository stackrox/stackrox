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
import useRestMutation from 'hooks/useRestMutation';
import useRestQuery from 'hooks/useRestQuery';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import BaseImagesModal from './BaseImagesModal';
import BaseImagesTable from './BaseImagesTable';

/**
 * Page component for managing base images. Displays a list of approved base images
 * and provides functionality to add and delete base images.
 */
function BaseImagesPage() {
    const [isAddModalOpen, setIsAddModalOpen] = useState(false);

    // Fetch base images on component mount
    const baseImagesRequest = useRestQuery(getBaseImages);

    // Delete mutation that refetches the list after successful deletion to keep UI in sync
    const deleteBaseImageMutation = useRestMutation((id: string) => deleteBaseImageFn(id), {
        onSuccess: () => {
            baseImagesRequest.refetch();
        },
    });

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
                    <Button variant="primary" onClick={() => setIsAddModalOpen(true)}>
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
                    onRemove={(baseImage) => deleteBaseImageMutation.mutate(baseImage.id)}
                    isRemoveInProgress={deleteBaseImageMutation.isLoading}
                    isLoading={baseImagesRequest.isLoading && !baseImagesRequest.data}
                    error={baseImagesRequest.error as Error | null}
                />
            </PageSection>
            <BaseImagesModal
                isOpen={isAddModalOpen}
                onClose={() => setIsAddModalOpen(false)}
                onSuccess={baseImagesRequest.refetch}
            />
        </>
    );
}

export default BaseImagesPage;
