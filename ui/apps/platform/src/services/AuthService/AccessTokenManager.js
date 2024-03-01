import EventEmitter from 'events';
/* eslint-disable import/no-duplicates */
import differenceInMilliSeconds from 'date-fns/difference_in_milliseconds';
import subSeconds from 'date-fns/sub_seconds';
import RefreshTokenTimeout from './RefreshTokenTimeout';

/**
 * Token and its expiry date
 * @typedef {Object} TokenInfo
 * @property {string} refreshUrl - Refresh URL to be used to refresh the access token
 * @property {string} expiry - ISO formatted (ISO 8601) string for the token expiration date
 */

/**
 * Token and its info (we keep them separate since token info maybe updated independently)
 * @typedef {Object} TokenWithInfo
 * @property {string} token - Token
 * @property {TokenInfo} info - Token info
 */

/**
 * This function is called to refresh the token.
 * @callback RefreshTokenFunc
 * @returns {Promise<TokenWithInfo>} Promise resolves to a new token and its info (expiration etc.)
 */

/**
 * An indicator if refresh token operation is in progress. If the operation that is in progress,
 * this Promise is always resolved (never rejected) regardless if token was refreshed or not.
 * @typedef {Promise<void>} RefreshTokenOpPromise
 */

/**
 * Listener that gets called when token refresh operation has been started.
 * @callback RefreshTokenListener
 * @param {!RefreshTokenOpPromise} opPromise - Operation Promise
 */

/**
 * Performs all the operations for storing, accessing and refreshing access token.
 *
 * @class AccessTokenManager
 */
export default class AccessTokenManager {
    /**
     * Creates a new instance of the manager. Note: currently all instances share same local storage.
     * @constructor
     * @param {Object} [options] - Configuration options
     * @param {RefreshTokenFunc} [options.refreshToken] - Function to call to refresh the token
     */
    constructor(options = {}) {
        this.options = options;

        this.refreshTimeout = new RefreshTokenTimeout();
        this.refreshTokenOpPromise = null;
        this.refreshTokenSymbol = Symbol('Refresh Token');
        this.eventEmitter = new EventEmitter();

        this.dispatchResponsePromise = null;
        this.dispatchSymbol = Symbol('Dispatch auth response');
        this.dispatchEventEmitter = new EventEmitter();
    }

    /**
     * Refreshes the token, skips refreshing if one is already in progress.
     * @method
     * @returns {!RefreshTokenOpPromise} Promise for just started or already being in progress operation
     */
    refreshToken = () => {
        if (this.refreshTokenOpPromise) {
            return this.refreshTokenOpPromise;
        } // already refreshing
        this.refreshTimeout.clear();

        if (!this.options.refreshToken) {
            return Promise.resolve();
        } // nothing to do, operation not started

        this.refreshTokenOpPromise = this.options
            .refreshToken()
            .then(() => {
                this.refreshTokenOpPromise = null;
            })
            .catch(() => {
                this.refreshTokenOpPromise = null;
            });
        this.eventEmitter.emit(this.refreshTokenSymbol, this.refreshTokenOpPromise);
        // Do a refresh every hour. For now this is statically defined, since we do not have access to the
        // underlying token and its expiration time.
        const expiryDate = subSeconds(new Date(), 3600);
        const delay = differenceInMilliSeconds(expiryDate, Date.now());
        this.refreshTimeout.set(this.refreshToken, delay);

        return this.refreshTokenOpPromise;
    };

    /**
     * Registers a listener that is called whenever refresh token operation starts.
     * @method
     * @param {!RefreshTokenListener} listener - Callback function
     */
    onRefreshTokenStarted = (listener) => {
        this.eventEmitter.on(this.refreshTokenSymbol, listener);
    };

    /**
     * Removes previously registered listener for the refresh token operation.
     * @method
     * @param {!RefreshTokenListener} listener - Callback function
     */
    removeRefreshTokenListener = (listener) => {
        this.eventEmitter.removeListener(this.refreshTokenSymbol, listener);
    };

    /**
     * Returns promise for refresh token operation or `null` if token isn't being refreshed.
     * @method
     * @returns {?RefreshTokenOpPromise} Promise or `null`
     */
    getRefreshTokenOpPromise = () => this.refreshTokenOpPromise;

    onDispatchResponseStarted = () => {
        this.dispatchResponsePromise = new Promise((resolve) => {
            this.dispatchEventEmitter.on(this.dispatchSymbol, resolve);
        });
    };

    getDispatchResponsePromise = () => this.dispatchResponsePromise;

    onDispatchResponseFinished = () => {
        this.dispatchEventEmitter.emit(this.dispatchSymbol);
        this.dispatchResponsePromise = null;
    };
}
