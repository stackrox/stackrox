import { applyMiddleware, compose, createStore } from 'redux';
import { routerMiddleware } from 'connected-react-router';

import createSagaMiddleware from 'redux-saga';
import createRavenMiddleware from 'raven-for-redux';
import Raven from 'raven-js';
import thunk from 'redux-thunk';

import rootSaga from 'sagas';
import createRootReducer from 'reducers';
import { actions as authActions } from 'reducers/auth';
import { addAuthInterceptors as authServiceAddAuthInterceptors } from 'services/AuthService';
import registerServerErrorHandler from 'services/serverErrorHandler';

const sagaMiddleware = createSagaMiddleware({
    onError: (error) => Raven.captureException(error),
});

// Omit Redux state to reduce size of payload in /api/logimbue request.

const ravenMiddleware = createRavenMiddleware(Raven, { stateTransformer: () => null });

export default function configureStore(initialState = {}, history) {
    const middlewares = [sagaMiddleware, routerMiddleware(history), ravenMiddleware, thunk];
    const enhancers = [applyMiddleware(...middlewares)];

    const composeEnhancers =
        process.env.NODE_ENV !== 'production' &&
        typeof window === 'object' &&
        window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
            ? window.__REDUX_DEVTOOLS_EXTENSION_COMPOSE__({ shouldHotReload: false })
            : compose;

    const rootReducer = createRootReducer(history);

    const store = createStore(rootReducer, initialState, composeEnhancers(...enhancers));

    authServiceAddAuthInterceptors((error) =>
        store.dispatch(authActions.handleAuthHttpError(error))
    );

    registerServerErrorHandler(
        () => store.dispatch({ type: 'serverStatus/RESPONSE_SUCCESS' }),
        () => store.dispatch({ type: 'serverStatus/RESPONSE_FAILURE', now: Date.now() })
    );

    sagaMiddleware.run(rootSaga);

    return store;
}
