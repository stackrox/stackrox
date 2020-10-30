import { saveFile } from 'services/DownloadService';

/**
 * Downloads diagnostic zip.
 * @param {string} queryString (assume it includes initial ? if non-empty)
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadDiagnostics(queryString) {
    return saveFile({
        method: 'get',
        url: `/api/extensions/diagnostics${queryString || ''}`,
    });
}
