import { take, spawn } from 'redux-saga/effects';
import { LOCATION_CHANGE } from 'react-router-redux';
import { matchPath } from 'react-router';

/**
 * Redux Saga helper function that spanws a provided saga on each location change action
 * (see [react-router-redux](https://github.com/ReactTraining/react-router/tree/master/packages/react-router-redux)
 * that matches the specified route (see [matchPath](https://reacttraining.com/react-router/web/api/matchPath)).
 *
 * @param {string|Object} route same as `matchPath` expects as a second argument
 * @param saga saga to be spawned on match, the output of `matchPath` will be passed as a first argument
 */
export default function* watchLocation(route, saga) {
    while (true) {
        const action = yield take(LOCATION_CHANGE);
        const { payload: location } = action;
        if (location && location.pathname) {
            const match = matchPath(location.pathname, route);
            if (match) {
                yield spawn(saga, match);
            }
        }
    }
}
