import { Bullseye } from '@patternfly/react-core';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import type { BaseImage } from 'services/BaseImagesService';

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
        <div>
            <p>BaseImagesTable placeholder</p>
            <p>Items: {baseImages.length}</p>
            <p>Is removing: {isRemoveInProgress ? 'yes' : 'no'}</p>
        </div>
    );
}

export default BaseImagesTable;
