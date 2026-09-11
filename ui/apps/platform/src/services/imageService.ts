import queryString from 'qs';

import type { ListImage, WatchedImage } from 'types/image.proto';
import type { ApiSortOption } from 'types/search';
import type { EmbeddedVulnerability } from 'types/vulnerability.proto';
import { getPaginationParams } from 'utils/searchUtils';

import axios from './instance';
import type { Empty } from './types';

const imagesUrl = '/v1/images';
const imagesCountUrl = '/v1/imagescount';
const watchedImagesUrl = '/v1/watchedimages';

export type ImageDetails = {
    id: string;
    name?: {
        remote?: string;
        fullName?: string;
    };
    priority?: string | number;
    scan?: {
        components?: {
            vulns?: EmbeddedVulnerability[];
        }[];
    };
};

export function fetchImageCount(query = ''): Promise<number> {
    const params = queryString.stringify({ query }, { arrayFormat: 'repeat' });
    return axios
        .get<{ count: number }>(`${imagesCountUrl}?${params}`)
        .then((response) => response.data?.count ?? 0);
}

export function listImages({
    query = '',
    sortOption,
    page = 1,
    perPage = 6,
}: {
    query?: string;
    sortOption: ApiSortOption;
    page?: number;
    perPage?: number;
}): Promise<ListImage[]> {
    const params = queryString.stringify(
        {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        { allowDots: true }
    );
    return axios
        .get<{ images: ListImage[] }>(`${imagesUrl}?${params}`)
        .then((response) => response.data?.images ?? []);
}

export function getImage(id: string): Promise<ImageDetails> {
    return axios
        .get<ImageDetails>(`${imagesUrl}/${encodeURIComponent(id)}`, {
            params: { strip_description: true },
        })
        .then((response) => response.data);
}

/*
 * Get array of images.
 *
 * Because GraphQL query has superseded this request except for PolicyScopeForm,
 * omit arguments for query params: search filter, sort options, page offset, page size.
 */
export function getImages(): Promise<ListImage[]> {
    return axios
        .get<{ images: ListImage[] }>(imagesUrl)
        .then((response) => response.data?.images ?? []);
}

/*
 * Get array of watched images identified by name.
 */
export function getWatchedImages(): Promise<WatchedImage[]> {
    return axios
        .get<{ watchedImages: WatchedImage[] }>(watchedImagesUrl)
        .then((response) => response.data.watchedImages ?? []);
}

/*
 * Stop watching an image identified by name.
 */
export function unwatchImage(name: string): Promise<Empty> {
    return axios
        .delete<Empty>(`${watchedImagesUrl}?name=${name}`)
        .then((response) => response.data);
}

/*
 * Start watching an image fully-qualified image, even if inactive, identified by name.
 *
 * The name of the image must be fully qualified, including a tag, but must NOT include a SHA.
 */
export function watchImage(name: string): Promise<string> {
    const requestPayload = {
        name,
    };
    const options = {
        // longer timeout needed to wait for pull and scan
        timeout: 300000, // 5 minutes is max for Chrome
    };

    return axios
        .post<WatchImageResponse>(watchedImagesUrl, requestPayload, options)
        .then((response) => {
            const { normalizedName, errorType, errorMessage } = response.data;
            if (errorType !== 'NO_ERROR') {
                throw new Error(errorMessage);
            }

            return normalizedName;
        });
}

export type WatchImageResponse = {
    // If the image was scanned successfully, this returns the normalized name of the image.
    // This depends on what we get from the registry.
    // For example, "docker.io/wordpress:latest" -> "docker.io/library/wordpress:latest"
    normalizedName: string;

    errorType: WatchImageErrorType;

    // Only set if error_type is NOT equal to "NO_ERROR".
    errorMessage: string; // TODO empty if not set?
};

export type WatchImageErrorType =
    | 'NO_ERROR'
    | 'INVALID_IMAGE_NAME'
    | 'NO_VALID_INTEGRATION'
    | 'SCAN_FAILED';
