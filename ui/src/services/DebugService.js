import { saveFile } from 'services/DownloadService';

/**
 * Downloads diagnostic zip.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadDiagnostics() {
    return saveFile({
        method: 'get',
        url: '/api/extensions/diagnostics'
    });
}
