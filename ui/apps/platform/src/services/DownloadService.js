import FileSaver from 'file-saver';
import axios from './instance';

/**
 * Common download service to download different types of files.
 * By default, timeout for downloads is removed. To override this behaviour, use timeout parameter.
 */
export function saveFile({ method, url, data, name = '', timeout = 0 }) {
    const options = {
        method,
        url,
        data,
        responseType: 'arraybuffer',
        name,
        timeout,
    };
    return axios(options)
        .then((response) => {
            if (response.data) {
                const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;
                const matches = filenameRegex.exec(response.headers['content-disposition']);

                const file = new Blob([response.data], {
                    type: response.headers['content-type'],
                });

                if (name && typeof name === 'string') {
                    FileSaver.saveAs(file, name);
                } else if (matches !== null && matches[1]) {
                    FileSaver.saveAs(file, matches[1].replace(/['"]/g, ''));
                } else {
                    throw new Error('Unable to extract file name');
                }
            } else {
                throw new Error('Expected response to contain "data" property');
            }
        })
        .catch((err) => {
            // because the responseType of the request is `arraybuffer`,
            // any error message is also wrapped in an ArrayBuffer data structure
            // we try to parse that to a string
            const parsedError = new TextDecoder().decode(err?.response?.data);

            // pass along the parsed error message, unless the parsing returned nothing
            throw parsedError || err;
        });
}
