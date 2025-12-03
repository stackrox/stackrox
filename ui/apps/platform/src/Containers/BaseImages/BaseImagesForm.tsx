import type { AxiosError } from 'axios';
import type { AddBaseImageRequest } from 'services/BaseImagesService';

export type BaseImagesFormProps = {
    onAddBaseImage: (request: AddBaseImageRequest) => void;
    isSubmitting: boolean;
    error: AxiosError | null;
};

function BaseImagesForm({ onAddBaseImage, isSubmitting, error }: BaseImagesFormProps) {
    return (
        <div>
            <p>BaseImagesForm placeholder</p>
            <p>Is submitting: {isSubmitting ? 'yes' : 'no'}</p>
            <p>Error: {error ? 'yes' : 'no'}</p>
            <button
                type="button"
                onClick={() => {
                    const request: AddBaseImageRequest = {
                        baseImageRepoPath: 'test',
                        baseImageTagPattern: 'test',
                    };
                    onAddBaseImage(request);
                }}
            >
                Test Add
            </button>
        </div>
    );
}

export default BaseImagesForm;
