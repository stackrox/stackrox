import { createStore, applyMiddleware, compose } from 'redux';
import { routerMiddleware } from 'react-router-redux';
import createSagaMiddleware from 'redux-saga';

import rootSaga from 'sagas';
import rootReducer from 'reducers';
import { actions as authActions } from 'reducers/auth';
import * as AuthService from 'services/AuthService';

const sagaMiddleware = createSagaMiddleware();

export default function configureStore(initialState = {}, history) {
    const middlewares = [sagaMiddleware, routerMiddleware(history)];
    if (process.env.NODE_ENV !== 'production') {
        // disable ESLint for next line since we need to make dev only dependency import
        // eslint-disable-next-line
        middlewares.push(require('redux-immutable-state-invariant').default());
    }
    const enhancers = [applyMiddleware(...middlewares)];

    // If Redux DevTools Extension is installed use it, otherwise use Redux compose
    /* eslint-disable no-underscore-dangle */
    const composeEnhancers =
        process.env.NODE_ENV !== 'production' &&
        typeof window === 'object' &&
        window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
            ? window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__({
                  // TODO Try to remove when `react-router-redux` is out of beta, LOCATION_CHANGE should not be fired more than once after hot reloading
                  // Prevent recomputing reducers for `replaceReducer`
                  shouldHotReload: false
              })
            : compose;
    /* eslint-enable */
    const store = createStore(rootReducer, initialState, composeEnhancers(...enhancers));

    // add auth interceptors before any HTTP request to APIs (i.e. before running sagas)
    AuthService.addAuthInterceptors(error =>
        store.dispatch(authActions.handleAuthHttpError(error))
    );
    sagaMiddleware.run(rootSaga);
    return store;
}
