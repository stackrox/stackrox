export type Risk = {
    id: string;
    subject: RiskSubject;
    score: number; // float
    results: RiskResult[];
};

export type RiskResult = {
    name: string;
    factors: RiskFactor[];
    score: number; // float
};

export type RiskFactor = {
    message: string;
    url: string;
};

export type RiskSubject = {
    id: string;
    namespace: string;
    clusterId: string;
    type: RiskSubjectType;
};

export type RiskSubjectType =
    | 'UNKNOWN'
    | 'DEPLOYMENT'
    | 'NAMESPACE'
    | 'CLUSTER'
    | 'NODE'
    | 'NODE_COMPONENT'
    | 'IMAGE'
    | 'IMAGE_COMPONENT'
    | 'SERVICEACCOUNT';
