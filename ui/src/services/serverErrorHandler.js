import axios from './instance';

let interceptorAdded = false;

export default function registerServerErrorHandler(successCallback, errorCallback) {
    if (interceptorAdded) return;
    axios.interceptors.response.use(
        response => {
            successCallback();
            return response;
        },
        error => {
            const status = error && error.response && error.response.status;
            if (!status || (status >= 502 && status <= 504)) {
                errorCallback();
            }
            return Promise.reject(error);
        }
    );
    interceptorAdded = true;
}
