const filenameSanitizerRegex = new RegExp('(:)|(/)|(\\s)', 'gi');

export function sanitizeFilename(filename: string, replacementChar: string = '_') {
    return filename.replaceAll(filenameSanitizerRegex, replacementChar);
}
