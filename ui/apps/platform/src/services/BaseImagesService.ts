import type { SlimUser } from 'types/user.proto';
import axios from './instance';
import type { Empty } from './types';

const baseImagesUrl = '/v2/baseimages';

export type BaseImageReference = {
    id: string;
    baseImageRepoPath: string;
    baseImageTagPattern: string;
    user: SlimUser;
};

export type BaseImagesResponse = {
    baseImageReferences: BaseImageReference[];
};

export type CreateBaseImageReferenceResponse = {
    baseImageReference: BaseImageReference;
};

/**
 * Fetch the list of configured base images.
 */
export function getBaseImages(): Promise<BaseImageReference[]> {
    return axios
        .get<BaseImagesResponse>(baseImagesUrl)
        .then((response) => response.data.baseImageReferences ?? []);
}

/**
 * Add a new base image to the system.
 */
export function addBaseImage(
    baseImageRepoPath: string,
    baseImageTagPattern: string
): Promise<BaseImageReference> {
    return axios
        .post<CreateBaseImageReferenceResponse>(baseImagesUrl, {
            baseImageRepoPath,
            baseImageTagPattern,
        })
        .then((response) => response.data.baseImageReference);
}

/**
 * Delete a base image from the system by ID.
 */
export function deleteBaseImage(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${baseImagesUrl}/${id}`).then((response) => response.data);
}

/**
 * Update the tag pattern of an existing base image.
 */
export function updateBaseImageTagPattern(id: string, baseImageTagPattern: string): Promise<Empty> {
    return axios
        .put<Empty>(`${baseImagesUrl}/${id}`, { baseImageTagPattern })
        .then((response) => response.data);
}
