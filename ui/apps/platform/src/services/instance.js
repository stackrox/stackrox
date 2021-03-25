// This is the one place where we're allowed to import directly from 'axios'.
// All other places must use the instance exported here.
// eslint-disable-next-line no-restricted-imports
import axios from 'axios';
import qs from 'qs';

import { ORCHESTRATOR_COMPONENT_KEY } from 'Containers/Navigation/OrchestratorComponentsToggle';

const instance = axios.create({
    timeout: 10000,
});

export const orchestratorQueryKey = 'orchestratorComponent';

export function appendOrchestratorComponentsQuery(url, showOrchestratorComponent) {
    const hasQuery = url.includes('?');
    const params = hasQuery ? url.split('?')[1] : null;
    // append orchestrator components query to URL
    const orchestratorComponentsQuery = {
        [orchestratorQueryKey]: showOrchestratorComponent,
    };
    const queryString = qs.stringify(orchestratorComponentsQuery);

    const delimiter = params ? '&' : '?';
    return `${url}${delimiter}${queryString}`;
}

instance.interceptors.request.use((config) => {
    // for openshift filterting toggle
    const showOrchestratorComponent = localStorage.getItem(ORCHESTRATOR_COMPONENT_KEY);
    if (showOrchestratorComponent === 'true') {
        const newConfig = { ...config };
        newConfig.url = appendOrchestratorComponentsQuery(config.url, showOrchestratorComponent);
        return newConfig;
    }
    return config;
});

export default instance;

// THE FOLLOWING CODE SNIPPET CAN BE USED TO DEBUG UNIT TESTS,
// IF YOU HAVEN'T MOCKED OUT AXIOS PROPERLY AND ARE GETTING
// CONSOLE ERRORS.
/*
export default {
    get: (url) => console.log('GET CALLED WITH', url),
    post: (url, data) => console.log('POST CALLED WITH', url, data),
    put: (url, data) => console.log('PUT CALLED WITH', url, data),
    patch: (url, data) => console.log('PATCH CALLED WITH', url, data),
    delete: (url) => console.log('DELETE CALLED WITH', url),
    interceptors: {
        request: {
            use: (args) => console.log('interceptors.request.use called with', args),
        },
        response: {
            use: (args) => console.log('interceptors.response.use called with', args),
        },
    },
};
*/
