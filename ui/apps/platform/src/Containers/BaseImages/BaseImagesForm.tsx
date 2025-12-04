import { Alert, Button, Flex, FlexItem } from '@patternfly/react-core';

import type { AddBaseImageRequest } from 'services/BaseImagesService';

export type BaseImagesFormProps = {
    onAddBaseImage: (request: AddBaseImageRequest) => void;
    isSubmitting: boolean;
};

function BaseImagesForm({ onAddBaseImage, isSubmitting }: BaseImagesFormProps) {
    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsMd' }}>
            <FlexItem>
                <Alert variant="info" isInline title="Form coming soon" component="p">
                    Will add baseImageRepoPath and baseImageTagPattern inputs
                </Alert>
            </FlexItem>
            <FlexItem>
                <Button
                    onClick={() =>
                        onAddBaseImage({ baseImageRepoPath: '', baseImageTagPattern: '' })
                    }
                    isLoading={isSubmitting}
                    isDisabled={isSubmitting}
                >
                    Save
                </Button>
            </FlexItem>
        </Flex>
    );
}

export default BaseImagesForm;
