import { saveFile } from 'services/DownloadService';

export type DiagnosticBundleRequest = {
    filterByClusters: string[];
    filterByStartingTime: string;
};

/**
 * Downloads diagnostic zip.
 * @param {string} queryString (assume it includes initial ? if non-empty)
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadDiagnostics(queryString: string) {
    return saveFile({
        method: 'get',
        url: `/api/extensions/diagnostics${queryString || ''}`,
        data: null,
        timeout: 300000, // setting 5 minutes as a default timeout value for diagnostic bundle
    });
}
