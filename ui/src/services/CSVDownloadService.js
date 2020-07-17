import { saveFile } from 'services/DownloadService';
import { addBrandedTimestampToString } from 'utils/dateUtils';

/**
 * Downloads CSV files
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadCSV(fileName, downloadUrl, queryString = '') {
    const url = queryString?.length ? `${downloadUrl}?${queryString}` : downloadUrl;
    const csvFileName = `${addBrandedTimestampToString(fileName)}.csv`;

    return saveFile({
        method: 'GET',
        url,
        data: null,
        name: csvFileName,
    });
}
