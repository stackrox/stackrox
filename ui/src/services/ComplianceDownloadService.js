import { saveFile } from 'services/DownloadService';

/**
 * Downloads CSV files for Compliance.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadCsv(params, fileName, downloadUrl) {
    const queryString = Object.keys(params)
        .map(key => `${key}=${params[key]}`)
        .join('&');
    const url = queryString !== '' ? `${downloadUrl}?${queryString}` : downloadUrl;
    return saveFile({
        method: 'get',
        url: `${url}`,
        data: null,
        name: `${fileName}.csv`
    });
}
