import FileSaver from 'file-saver';
import { sanitizeFilename } from 'utils/fileUtils';

import axios from './instance';

export function generateAndSaveSbom({ imageName }: { imageName: string }): Promise<void> {
    return axios({
        method: 'POST',
        url: '/api/v1/images/sbom',
        data: { imageName },
        timeout: 0,
    }).then((response) => {
        const fileName = sanitizeFilename(`${imageName}.sbom`);
        const file = new Blob([response.data], {
            type: response.headers['content-type'],
        });

        FileSaver.saveAs(file, fileName);
    });
}
