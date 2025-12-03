import type { SlimUser } from 'types/user.proto';
import axios from './instance';
import type { Empty } from './types';

const baseImagesUrl = '/v2/baseimages';

export type BaseImage = {
    id: string;
    baseImageRepoPath: string;
    baseImageTagPattern: string;
    user: SlimUser;
};

export type AddBaseImageRequest = {
    baseImageRepoPath: string;
    baseImageTagPattern: string;
};

export type BaseImagesResponse = {
    baseImages: BaseImage[];
};

/**
 * Fetch the list of configured base images.
 */
export function getBaseImages(): Promise<BaseImage[]> {
    // TODO: Replace with actual API call once backend is ready
    // return axios
    //     .get<BaseImagesResponse>(baseImagesUrl)
    //     .then((response) => response.data.baseImages ?? []);
    return Promise.resolve([]);
}

/**
 * Add a new base image to the system.
 */
export function addBaseImage(request: AddBaseImageRequest): Promise<BaseImage> {
    return axios.post<BaseImage>(baseImagesUrl, request).then((response) => response.data);
}

/**
 * Delete a base image from the system by ID.
 */
export function deleteBaseImage(id: string): Promise<Empty> {
    return axios.delete<Empty>(`${baseImagesUrl}/${id}`).then((response) => response.data);
}
