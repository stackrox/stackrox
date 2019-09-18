import Raven from 'raven-js';

const logError = error => {
    Raven.captureException(error);
    if (process.env.NODE_ENV === 'development') {
        // eslint-disable-next-line
        console.error(error);
    }
};

export default logError;
