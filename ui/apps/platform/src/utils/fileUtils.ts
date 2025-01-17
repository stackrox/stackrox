import { AxiosResponse } from 'axios';

const filenameSanitizerRegex = new RegExp('(:)|(/)|(\\s)', 'gi');

export function sanitizeFilename(filename: string, replacementChar: string = '_') {
    return filename.replaceAll(filenameSanitizerRegex, replacementChar);
}

const filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;

export function parseAxiosResponseAttachment(response: AxiosResponse): {
    file: Blob;
    filename: string | null;
} {
    const matches = filenameRegex.exec(response.headers['content-disposition'] ?? '');
    const filename = matches && matches[1] ? matches[1] : null;
    const file = new Blob([response.data], { type: response.headers['content-type'] });
    return { file, filename };
}
