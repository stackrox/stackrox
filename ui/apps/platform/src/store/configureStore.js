import { createStore, applyMiddleware, compose } from 'redux';
import { routerMiddleware } from 'connected-react-router';
import createSagaMiddleware from 'redux-saga';
import createRavenMiddleware from 'raven-for-redux';
import Raven from 'raven-js';

import rootSaga from 'sagas';
import createRootReducer from 'reducers';
import { actions as authActions } from 'reducers/auth';
import * as AuthService from 'services/AuthService';
import { actions as serverErrorActions } from 'reducers/serverError';
import registerServerErrorHandler from 'services/serverErrorHandler';

const sagaMiddleware = createSagaMiddleware({
    onError: (error) => Raven.captureException(error),
});

// Omit Redux state to reduce size of payload in /api/logimbue request.
const ravenMiddleware = createRavenMiddleware(Raven, { stateTransformer: () => null });

export default function configureStore(initialState = {}, history) {
    const middlewares = [sagaMiddleware, routerMiddleware(history), ravenMiddleware];
    const enhancers = [applyMiddleware(...middlewares)];

    // If Redux DevTools Extension is installed use it, otherwise use Redux compose
    const composeEnhancers =
        process.env.NODE_ENV !== 'production' &&
        typeof window === 'object' &&
        window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
            ? window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__({
                  // TODO Try to remove when `react-router-redux` is out of beta, LOCATION_CHANGE should not be fired more than once after hot reloading
                  // Prevent recomputing reducers for `replaceReducer`
                  shouldHotReload: false,
              })
            : compose;
    const rootReducer = createRootReducer(history);
    const store = createStore(rootReducer, initialState, composeEnhancers(...enhancers));

    // add auth interceptors before any HTTP request to APIs (i.e. before running sagas)
    AuthService.addAuthInterceptors((error) =>
        store.dispatch(authActions.handleAuthHttpError(error))
    );

    registerServerErrorHandler(
        () => store.dispatch(serverErrorActions.recordServerSuccess()),
        () => store.dispatch(serverErrorActions.recordServerError())
    );
    sagaMiddleware.run(rootSaga);
    return store;
}
