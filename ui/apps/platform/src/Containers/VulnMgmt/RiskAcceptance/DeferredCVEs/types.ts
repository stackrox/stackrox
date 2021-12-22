export type DeferredCVEsToBeAssessed = {
    type: 'DEFERRAL';
    action: 'UNDO';
    requestIDs: string[];
} | null;
