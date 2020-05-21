import { saveFile } from 'services/DownloadService';
import { addBrandedTimestampToString } from 'utils/dateUtils';
import queryService from 'utils/queryService';

/**
 * Downloads CSV files
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadCSV(fileName, downloadUrl, params = {}) {
    const queryString = queryService.objectToWhereClause(params, '&');
    const url = queryString !== '' ? `${downloadUrl}?query=${queryString}` : downloadUrl;
    const csvFileName = `${addBrandedTimestampToString(fileName)}.csv`;

    return saveFile({
        method: 'GET',
        url,
        data: null,
        name: csvFileName,
    });
}
