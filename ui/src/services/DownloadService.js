import FileSaver from 'file-saver';
import axios from './instance';

// for now only one function exported from this file
/* eslint-disable import/prefer-default-export */

/**
 * Common download service to download different types of files.
 */
export function saveFile({ method, url, data, name }) {
    const options = {
        method,
        url,
        data,
        responseType: 'arraybuffer',
        name
    };
    return axios(options).then(response => {
        if (response.data) {
            const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;
            const matches = filenameRegex.exec(response.headers['content-disposition']);

            const file = new Blob([response.data], {
                type: response.headers['content-type']
            });

            if (name !== '') {
                FileSaver.saveAs(file, name);
            } else if (matches !== null && matches[1]) {
                FileSaver.saveAs(file, matches[1].replace(/['"]/g, ''));
            } else {
                throw new Error('Unable to extract file name');
            }
        } else {
            throw new Error('Expected response to contain "data" property');
        }
    });
}
