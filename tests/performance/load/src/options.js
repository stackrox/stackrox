export const defaultOptions = {
    insecureSkipTLSVerify: true,
    setupTimeout: '300s',
    thresholds: {
        // Remove all http calls from lib.
        'http_req_duration{lib:true}': [`max>=0`],
    },
};
