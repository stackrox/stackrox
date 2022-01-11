export type FalsePositiveCVEsToBeAssessed = {
    type: 'FALSE_POSITIVE';
    action: 'UNDO';
    requestIDs: string[];
} | null;
