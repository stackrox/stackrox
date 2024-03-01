// eslint-disable-next-line no-restricted-imports, import/no-named-default
import { default as axiosGlobal } from 'axios';
import addTokenRefreshInterceptors, {
    doNotStallRequestConfig,
} from './addTokenRefreshInterceptors';
import AccessTokenManager from './AccessTokenManager';

function newAxiosInstance() {
    const instance = axiosGlobal.create();
    instance.request = jest.fn(); // to make sure we're not making any requests
    return instance;
}

function addInterceptorAndGetHandler(
    axios = newAxiosInstance(),
    accessTokenManager = new AccessTokenManager(),
    options = {}
) {
    addTokenRefreshInterceptors(axios, accessTokenManager, options);
    return axios.interceptors.response.handlers[0];
}

describe('addTokenRefreshInterceptors', () => {
    it('should allow success responses to go through', () => {
        const handler = addInterceptorAndGetHandler();
        expect(handler.fulfilled({ data: 'data' })).toEqual({ data: 'data' });
    });

    it('should reject non 401 responses', () => {
        const handler = addInterceptorAndGetHandler();
        const error = {
            response: {
                status: 403,
            },
        };
        return expect(handler.rejected(error)).rejects.toMatchObject(error);
    });

    it('should retry failed request after ongoing token refresh is finished', () => {
        const axios = newAxiosInstance();
        const m = new AccessTokenManager();
        m.getRefreshTokenOpPromise = jest.fn().mockResolvedValue();
        m.refreshToken = jest.fn().mockResolvedValue();
        axios.request = (config) => Promise.resolve(config);
        const handler = addInterceptorAndGetHandler(axios, m);

        const error = {
            response: { status: 401 },
            config: { myRequest: true },
        };
        return expect(handler.rejected(error))
            .resolves.toMatchObject({ myRequest: true })
            .then(() => {
                // should not initiate refresh token operation as another one is in progress
                expect(m.refreshToken).not.toHaveBeenCalled();
            });
    });

    it('should retry failed request if dispatch response is finished', () => {
        const axios = newAxiosInstance();
        axios.request = (config) => Promise.resolve(config);
        const m = new AccessTokenManager();
        m.getDispatchResponsePromise = jest.fn().mockResolvedValue();
        m.refreshToken = jest.fn().mockResolvedValue();

        const handler = addInterceptorAndGetHandler(axios, m);

        const error = {
            response: { status: 401 },
            config: { myRequest: true },
        };
        return expect(handler.rejected(error))
            .resolves.toMatchObject({ myRequest: true })
            .then(() => {
                expect(m.refreshToken).not.toHaveBeenCalled();
            });
    });

    it('should initiate token refresh in case of 401 failure', () => {
        const axios = newAxiosInstance();
        axios.request = (config) => Promise.resolve(config);
        const m = new AccessTokenManager();
        m.refreshToken = jest.fn().mockResolvedValue();

        const handler = addInterceptorAndGetHandler(axios, m);
        const error = {
            response: { status: 401 },
            config: { myRequest: true },
        };
        return expect(handler.rejected(error))
            .resolves.toMatchObject({ myRequest: true })
            .then(() => {
                expect(m.refreshToken).toHaveBeenCalledTimes(1);
            });
    });

    it('should call error callback when retry / token refresh did not help', async () => {
        const axios = newAxiosInstance();
        axios.request = (config) =>
            Promise.resolve({
                response: { status: 401 },
                config,
            });
        const m = new AccessTokenManager();
        m.refreshToken = jest.fn().mockResolvedValue();
        const handleAuthError = jest.fn();

        const handler = addInterceptorAndGetHandler(axios, m, { handleAuthError });

        const error = {
            response: { status: 401 },
            config: { myRequest: true },
        };
        const retriedResponseError = await handler.rejected(error);
        return expect(handler.rejected(retriedResponseError))
            .rejects.toEqual(retriedResponseError)
            .then(() => {
                expect(handleAuthError).toHaveBeenCalledTimes(1);
            });
    });

    it('should stall new requests while token is being refreshed', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const m = new AccessTokenManager({ refreshToken });
        const axios = newAxiosInstance();
        addTokenRefreshInterceptors(axios, m);
        m.refreshToken(); // start refreshing the token

        // request handler should get attached
        const handler = axios.interceptors.request.handlers[0];
        const request = { myRequest: true };
        const promise = handler.fulfilled(request);
        expect(promise.then).toEqual(expect.any(Function)); // indeed blocked on promise

        return expect(promise)
            .resolves.toMatchObject({ myRequest: true })
            .then(() => {
                // request handler should be detached at this point
                expect(axios.interceptors.request.handlers[0]).toBeFalsy();
            });
    });

    it('should not stall request marked for not stalling', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const m = new AccessTokenManager({ refreshToken });
        const axios = newAxiosInstance();
        addTokenRefreshInterceptors(axios, m);
        m.refreshToken(); // start refreshing the token

        const handler = axios.interceptors.request.handlers[0];
        const request = { myRequest: true, ...doNotStallRequestConfig };
        expect(handler.fulfilled(request)).toEqual(request); // not a promise
    });

    it('should not stall any requests if the corresponding config options is provided', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const m = new AccessTokenManager({ refreshToken });
        const axios = newAxiosInstance();
        addTokenRefreshInterceptors(axios, m, { doNotStallRequests: true });
        m.refreshToken(); // start refreshing the token

        // request interceptor isn't even attached
        expect(axios.interceptors.request.handlers[0]).toBeFalsy();
    });

    it('should detach the interceptors', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const m = new AccessTokenManager({ refreshToken });
        const axios = newAxiosInstance();
        const detach = addTokenRefreshInterceptors(axios, m);

        expect(axios.interceptors.response.handlers[0]).toBeTruthy();
        detach();
        expect(axios.interceptors.response.handlers[0]).toBeFalsy();

        // check that there is no listener on access manager either,
        // e.g. refreshing token won't attach request interceptor to stall requests
        m.refreshToken();
        expect(axios.interceptors.request.handlers[0]).toBeFalsy();
    });
});
