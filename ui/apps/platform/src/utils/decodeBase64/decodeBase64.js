/**
 * This function uses the atob function while considering some special characters that can't be
 * properly decoded using atob
 *
 * More info about how the backend uses URL/Filename safe encoding https://www.admfactory.com/base64-encode-golang/
 *
 * @param {String} str - A base64 encoded string
 * @returns {String}
 */
function decodeBase64(str) {
    return atob(str.replace(/_/g, '/').replace(/-/g, '+'));
}

export default decodeBase64;
