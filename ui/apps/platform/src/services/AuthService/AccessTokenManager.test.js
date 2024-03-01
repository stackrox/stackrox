import AccessTokenManager from './AccessTokenManager';

describe('AccessTokenManager', () => {
    beforeEach(() => {
        jest.useFakeTimers();
    });

    it('should invoke refresh token routine after 1 hour', () => {
        const refreshToken = jest.fn().mockResolvedValue();
        // eslint-disable-next-line no-unused-vars
        const m = new AccessTokenManager({ refreshToken });
        m.refreshToken();
        jest.advanceTimersByTime(3600030);
        expect(refreshToken).toHaveBeenCalled();
    });

    it('should notify attached listener about token being refreshed', () => {
        const refreshToken = jest.fn().mockRejectedValue();
        let opPromiseToTest = null;
        const refreshTokenListener = (opPromise) => {
            opPromiseToTest = opPromise;
        };
        const m = new AccessTokenManager({ refreshToken });
        m.onRefreshTokenStarted(refreshTokenListener);
        m.refreshToken();
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

    it('should resolve dispatch response', () => {
        const m = new AccessTokenManager();
        expect(m.getDispatchResponsePromise()).toBeNull();
        m.onDispatchResponseStarted();
        expect(m.getDispatchResponsePromise()).not.toBeNull();
        m.onDispatchResponseFinished();
        expect(m.getRefreshTokenOpPromise()).toBeNull();
    });
});
