import { combineReducers } from 'redux';

import bindSelectors from 'utils/bindSelectors';
import alerts, { selectors as alertSelectors } from './alerts';
import benchmarks, { selectors as benchmarkSelectors } from './benchmarks';
import clusters, { selectors as clusterSelectors } from './clusters';
import policies, { selectors as policySelectors } from './policies';
import route, { selectors as routeSelectors } from './routes';

// Reducers

const appReducer = combineReducers({
    alerts,
    benchmarks,
    clusters,
    policies
});

const rootReducer = combineReducers({
    route,
    app: appReducer
});

export default rootReducer;

// Selectors

const getRoute = state => state.route;
const getApp = state => state.app;
const getAlerts = state => getApp(state).alerts;
const getBenchmarks = state => getApp(state).benchmarks;
const getClusters = state => getApp(state).clusters;
const getPolicies = state => getApp(state).policies;

const boundSelectors = {
    ...bindSelectors(getAlerts, alertSelectors),
    ...bindSelectors(getBenchmarks, benchmarkSelectors),
    ...bindSelectors(getClusters, clusterSelectors),
    ...bindSelectors(getPolicies, policySelectors),
    ...bindSelectors(getRoute, routeSelectors)
};

export const selectors = {
    ...boundSelectors
};
