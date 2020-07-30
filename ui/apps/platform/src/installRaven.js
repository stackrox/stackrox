import Raven from 'raven-js';

import axios from './services/instance';

let ravenInstalled = false;

export default function installRaven() {
    if (ravenInstalled) {
        Raven.captureException(new Error('Raven is already installed'));
        return;
    }

    // since hosted or on-prem Sentry isn't being used, there is no configuration we should be doing,
    // but raven-js requires to have some DSN (see https://github.com/getsentry/raven-js/issues/999)
    Raven.config('https://fakeuser@noserver/stackrox').install();

    Raven.setTransport(({ data, onSuccess, onError }) => {
        axios.post('/api/logimbue', data).then(onSuccess, onError);
    });

    ravenInstalled = true;
}
