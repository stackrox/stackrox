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

/**
 * Fetch the list of configured base images.
 */
export function getBaseImages(): Promise<BaseImageReference[]> {
    // TODO: Replace with actual API call once backend is ready
    // return axios
    //     .get<BaseImagesResponse>(baseImagesUrl)
    //     .then((response) => response.data.baseImageReferences ?? []);
    return Promise.resolve([
        {
            id: '1',
            baseImageRepoPath: 'library/ubuntu',
            baseImageTagPattern: '20.04.*',
            user: { id: '1', username: 'admin', name: 'Admin User' },
        },
        {
            id: '2',
            baseImageRepoPath: 'library/alpine',
            baseImageTagPattern: '3.*',
            user: { id: '2', username: 'admin', name: 'Admin User' },
        },
    ]);
}

/**
 * Add a new base image to the system.
 */
export function addBaseImage(
    baseImageRepoPath: string,
    baseImageTagPattern: string
): Promise<BaseImageReference> {
    return axios
        .post<BaseImageReference>(baseImagesUrl, { baseImageRepoPath, baseImageTagPattern })
        .then((response) => response.data);
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
