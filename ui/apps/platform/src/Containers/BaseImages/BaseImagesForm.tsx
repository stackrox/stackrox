import type { AxiosError } from 'axios';
import { Alert } from '@patternfly/react-core';
import type { AddBaseImageRequest } from 'services/BaseImagesService';

export type BaseImagesFormProps = {
    onAddBaseImage: (request: AddBaseImageRequest) => void;
    isSubmitting: boolean;
    error: AxiosError | null;
};

// eslint-disable-next-line @typescript-eslint/no-unused-vars
function BaseImagesForm({ onAddBaseImage, isSubmitting, error }: BaseImagesFormProps) {
    return (
        <Alert variant="info" isInline title="Form coming soon" component="p">
            Will add baseImageRepoPath and baseImageTagPattern inputs
        </Alert>
    );
}

export default BaseImagesForm;
