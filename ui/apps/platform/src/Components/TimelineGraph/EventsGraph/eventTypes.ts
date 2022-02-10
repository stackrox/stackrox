export type Event = {
    id: string;
    name: string;
    args?: string;
    type: string;
    uid?: number;
    parentName?: string;
    parentUid?: number;
    reason?: string;
    timestamp: string;
    inBaseline?: boolean;
};
