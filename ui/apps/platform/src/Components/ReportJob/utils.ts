import { saveFile } from 'services/DownloadService';
import { sanitizeFilename } from 'utils/fileUtils';
import { jobContextTabs } from './types';
import type { JobContextTab } from './types';

export function ensureJobContextTab(value: unknown): JobContextTab {
    if (typeof value === 'string' && jobContextTabs.includes(value as JobContextTab)) {
        return value as JobContextTab;
    }
    return 'CONFIGURATION_DETAILS';
}

export function onDownloadReport({ reportJobId, name, completedAt, baseDownloadURL }) {
    const filename = `${name}-${completedAt}`;
    const sanitizedFilename = sanitizeFilename(filename);
    return saveFile({
        method: 'get',
        url: `${baseDownloadURL}?id=${reportJobId}`,
        data: null,
        timeout: 300000,
        name: `${sanitizedFilename}.zip`,
    });
}
