export const httpURLPattern = /(https?:\/\/[a-zA-Z0-9]+\.[^\s]{2,})/g;

const urlPattern = new RegExp(
    '^(https?:\\/\\/)?' + // protocol
    '((([a-z\\d]([a-z\\d-]*[a-z\\d])*)\\.)+[a-z]{2,}|' + // domain name
    '((\\d{1,3}\\.){3}\\d{1,3}))' + // OR ip (v4) address
    '(\\:\\d+)?(\\/[-a-z\\d%_.~+]*)*' + // port and path
    '(\\?[;&a-z\\d%_.~+=-]*)?' + // query string
        '(\\#[-a-z\\d_]*)?$',
    'i'
); // fragment locator

export function isValidURL(str) {
    return !!urlPattern.test(str);
}

export default {
    isValidURL
};
