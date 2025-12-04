import { Alert, Bullseye } from '@patternfly/react-core';

import type { BaseImage } from 'services/BaseImagesService';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';

export type BaseImagesTableProps = {
    baseImages: BaseImage[];
    onRemove: (baseImage: BaseImage) => void;
    isRemoveInProgress: boolean;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function BaseImagesTable({ baseImages, onRemove, isRemoveInProgress }: BaseImagesTableProps) {
    if (baseImages.length === 0) {
        return (
            <Bullseye>
                <EmptyStateTemplate title="No base images configured" headingLevel="h2">
                    Add your first base image to start tracking layer-specific vulnerabilities
                </EmptyStateTemplate>
            </Bullseye>
        );
    }

    return (
        <Alert variant="info" isInline title="Base images table coming soon" component="p">
            You have {baseImages.length} base image{baseImages.length !== 1 ? 's' : ''}. Table UI
            will be implemented soon.
        </Alert>
    );
}

export default BaseImagesTable;
