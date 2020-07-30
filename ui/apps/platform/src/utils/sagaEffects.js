import { take, fork } from 'redux-saga/effects';
import { LOCATION_CHANGE } from 'react-router-redux';
import { matchPath } from 'react-router-dom';

/**
 * The location match object.
 * @typedef {Object} LocationMatch
 * @property {Object} location - the matched location object
 * @property {Object} match - the output of `matchPath` (see [matchPath](https://reacttraining.com/react-router/web/api/matchPath))
 */

/**
 * Redux Saga effect that spawns a provided `saga` on each location change action
 * (see [react-router-redux](https://github.com/ReactTraining/react-router/tree/master/packages/react-router-redux)
 * that matches the specified `route` (see [matchPath](https://reacttraining.com/react-router/web/api/matchPath)).
 *
 * @param {string|Object} route same as `matchPath` expects as a second argument
 * @param saga saga (a Generator function)
 * @param args list or arguments to be passed to the started saga, the {@link LocationMatch} object will be added as the last argument
 */
export const takeEveryLocation = (route, saga, ...args) =>
    fork(function* worker() {
        while (true) {
            const action = yield take(LOCATION_CHANGE);
            const { payload: location } = action;
            if (location && location.pathname) {
                const match = matchPath(location.pathname, route);
                if (match) {
                    yield fork(saga, ...args.concat({ match, location }));
                }
            }
        }
    });

/**
 * Redux Saga effect that spawns a provided `saga` on each location change action
 * (see [react-router-redux](https://github.com/ReactTraining/react-router/tree/master/packages/react-router-redux),
 * that has the location matching the specified `route` (see [matchPath](https://reacttraining.com/react-router/web/api/matchPath))
 * and there was no match for the previous location change (in other words multiple location changes happen in a row and all of
 * them match the specified `route`, only for the first such location a provided `saga` will be started).
 *
 * @param {string|Object} route same as `matchPath` expects as a second argument
 * @param saga saga (a Generator function)
 * @param args list or arguments to be passed to the started saga, the {@link LocationMatch} object will be added as the last argument
 */
export const takeEveryNewlyMatchedLocation = (route, saga, ...args) =>
    fork(function* worker() {
        let prevLocationMatched = false;
        while (true) {
            const action = yield take(LOCATION_CHANGE);
            const { payload: location } = action;
            if (location && location.pathname) {
                const match = matchPath(location.pathname, route);
                if (match && !prevLocationMatched) {
                    yield fork(saga, ...args.concat({ match, location }));
                }
                prevLocationMatched = !!match;
            }
        }
    });
