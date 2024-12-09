import { saveFile } from './DownloadService';

export function generateAndSaveSbom({ imageName }: { imageName: string }): Promise<void> {
    return saveFile({
        method: 'post',
        url: '/api/v1/images/sbom',
        data: { imageName },
    });
}
