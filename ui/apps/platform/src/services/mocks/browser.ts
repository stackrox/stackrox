// eslint-disable-next-line import/no-extraneous-dependencies
import { setupWorker } from 'msw';
import { handlers } from './handlers';

export function startMockServiceWorker() {
    const worker = setupWorker(...handlers);
    return worker.start({
        onUnhandledRequest: 'bypass',
        serviceWorker: {
            // This is the default value of `url`, made explicit here for traceability
            url: '/mockServiceWorker.js',
        },
    });
}
