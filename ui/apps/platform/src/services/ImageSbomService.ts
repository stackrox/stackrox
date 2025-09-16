import FileSaver from 'file-saver';
import { parseAxiosResponseAttachment } from 'utils/fileUtils';

import axios from './instance';

export function generateAndSaveSbom({ imageName }: { imageName: string }): Promise<void> {
    return axios({
        method: 'POST',
        url: '/api/v1/images/sbom',
        data: { imageName },
        timeout: 0,
    }).then((response) => {
        const { filename } = parseAxiosResponseAttachment(response);
        const file = new Blob([JSON.stringify(response.data)], { type: 'application/json' });
        FileSaver.saveAs(file, filename);
    });
}
