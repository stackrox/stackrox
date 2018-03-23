import axios from 'axios';
import saveAs from 'file-saver';

/**
 * Common download service to download different types of files.
 */
export default function DownloadService({ url, data }) {
    const options = {
        method: 'post',
        url,
        data,
        responseType: 'arraybuffer'
    };
    axios(options)
        .then(response => {
            if (response.data) {
                const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;
                const matches = filenameRegex.exec(response.headers['content-disposition']);

                if (matches !== null && matches[1]) {
                    const filename = matches[1].replace(/['"]/g, '');
                    const file = new Blob([response.data], {
                        type: response.headers['content-type']
                    });
                    saveAs(file, filename);
                }
            }
        })
        .catch(error => {
            console.error(error);
        });
}
