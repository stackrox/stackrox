import axios from 'axios';

import AuthService from 'services/AuthService';

let requestTokenInterceptor = null;
let responseInterceptor = null;

const apiInterceptors = {
    addRequestTokenInterceptor: () => {
        if (requestTokenInterceptor) return;
        // Adds an interceptor for api requests
        requestTokenInterceptor = axios.interceptors.request.use(
            config => {
                // If there is a token available, then set the headers with that token
                const token = AuthService.getAccessToken();
                if (token) {
                    const newConfig = Object.assign({}, config);
                    newConfig.headers.Authorization = `Bearer ${token}`;
                    return newConfig;
                }
                return config;
            },
            error => Promise.reject(error)
        );
    },
    addResponseInterceptor: () => {
        if (responseInterceptor) return;
        // Adds an interceptor for api requests
        responseInterceptor = axios.interceptors.response.use(
            response => response,
            error => {
                // if user is unauthenticated then go to the Login page
                if (error.response.status === 403) window.location = '/login';
                if (error.response.status === 401) {
                    AuthService.logout();
                    window.location = '/login';
                }
                return Promise.reject(error);
            }
        );
    }
};

export default apiInterceptors;
