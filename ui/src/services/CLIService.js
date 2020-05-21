import { saveFile } from 'services/DownloadService';

/**
 * Downloads cluster YAML configuration.
 *
 * @returns {Promise<undefined, Error>} resolved if operation was successful
 */
export default function downloadCLI(type) {
    let name = 'roxctl';
    let suffix = type;
    if (type === 'windows') {
        name = 'roxctl.exe';
        suffix = 'windows.exe';
    }
    return saveFile({
        method: 'get',
        url: `/api/cli/download/roxctl-${suffix}`,
        name,
    });
}
