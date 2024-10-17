import { saveFile } from './DownloadService';

export function savePoliciesAsCustomResource(ids: string[]): Promise<void> {
    return saveFile({
        method: 'post',
        url: '/api/policy/custom-resource/save-as-zip',
        data: { ids },
    });
}
