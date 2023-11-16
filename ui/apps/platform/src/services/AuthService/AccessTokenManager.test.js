/* global global */
import AccessTokenManager from './AccessTokenManager';

describe('AccessTokenManager', () => {
    beforeEach(() => {
        jest.useFakeTimers();
    });

    it('should store and then return stored token', () => {
        const m = new AccessTokenManager();
        m.setToken('my-token');
        expect(m.getToken()).toBe('my-token');
    });

    it('should store token', () => {
        const m = new AccessTokenManager();
        m.setToken('my-token');
        expect(m.getToken()).toBe('my-token');
    });

    it('should store token info', () => {
        const m = new AccessTokenManager();
        m.setToken('my-token', { d: 'data' });
        expect(m.tokenInfo).toEqual({ d: 'data' });

        m.setToken('my-token');
        expect(m.tokenInfo).toEqual(null);

        m.updateTokenInfo({ d: 'data' });
        expect(m.tokenInfo).toEqual({ d: 'data' });
    });

    it('should clear stored token and its info', () => {
        const m = new AccessTokenManager();
        m.setToken('my-token', { d: 'data' });
        m.clearToken();
        expect(m.getToken()).toBe(null);
        expect(m.tokenInfo).toBe(null);
    });

    it('should invoke refresh token routine 30 sec before token expires', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const m = new AccessTokenManager({ refreshToken });
        const tokenInfo = { expiry: new Date(Date.now() + 31000).toISOString() };

        m.setToken('my-token', tokenInfo);
        expect(refreshToken).not.toHaveBeenCalled();
        jest.advanceTimersByTime(1000);
        expect(refreshToken).toHaveBeenCalledWith(tokenInfo);
    });

    it('should clear timeout on refresh token invocation', () => {
        const timeoutSpy = jest.spyOn(global, 'clearTimeout');
        const m = new AccessTokenManager();
        const tokenInfo = { expiry: new Date(Date.now() + 31000).toISOString() };
        m.setToken('my-token', tokenInfo);
        m.refreshToken();
        expect(timeoutSpy).toHaveBeenCalledTimes(1);
    });

    it('should store new token info after refresh', () => {
        const refreshToken = jest
            .fn()
            .mockResolvedValue({ token: 'my-token-2', info: { d: 'data' } });
        const m = new AccessTokenManager({ refreshToken });

        m.setToken('my-token');
        m.refreshToken();

        return m.getRefreshTokenOpPromise().then(() => {
            expect(m.getToken()).toEqual('my-token-2');
            expect(m.tokenInfo).toEqual({ d: 'data' });
        });
    });

    it('should notify attached listener about token being refreshed', () => {
        const refreshToken = jest.fn().mockRejectedValue();
        let opPromiseToTest = null;
        const refreshTokenListener = (opPromise) => {
            opPromiseToTest = opPromise;
        };
        const tokenInfo = { expiry: new Date(Date.now() + 31000).toISOString() };
        const m = new AccessTokenManager({ refreshToken });

        m.onRefreshTokenStarted(refreshTokenListener);
        m.setToken('my-token', tokenInfo);
        jest.runAllTimers();
        expect(m.getRefreshTokenOpPromise()).toEqual(opPromiseToTest);
        return expect(opPromiseToTest).resolves.toBe(undefined);
    });

    it('should not notify removed refresh token listener', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const refreshTokenListener = jest.fn();
        const m = new AccessTokenManager({ refreshToken });

        m.onRefreshTokenStarted(refreshTokenListener);
        m.refreshToken();
        expect(refreshTokenListener).toHaveBeenCalledTimes(1);

        m.removeRefreshTokenListener(refreshTokenListener);
        return m.getRefreshTokenOpPromise().then(() => {
            expect(m.getRefreshTokenOpPromise()).toEqual(null);
            m.refreshToken();
            expect(refreshToken).toHaveBeenCalledTimes(2);
            expect(refreshTokenListener).toHaveBeenCalledTimes(1);
        });
    });

    it('should not notify refresh token listener if previous token refresh is in progress', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        const refreshTokenListener = jest.fn();
        const m = new AccessTokenManager({ refreshToken });

        m.onRefreshTokenStarted(refreshTokenListener);
        m.refreshToken();
        expect(refreshTokenListener).toHaveBeenCalledTimes(1);

        const firstOpPromise = m.getRefreshTokenOpPromise();
        m.refreshToken();
        expect(refreshTokenListener).toHaveBeenCalledTimes(1);
        expect(m.getRefreshTokenOpPromise()).toEqual(firstOpPromise);
    });
});
