export type Taint = {
    key: string;
    value: string;
    taintEffect: TaintEffect;
};

export type TaintEffect =
    | 'UNKNOWN_TAINT_EFFECT'
    | 'NO_SCHEDULE_TAINT_EFFECT'
    | 'PREFER_NO_SCHEDULE_TAINT_EFFECT'
    | 'NO_EXECUTE_TAINT_EFFECT';

export type Toleration = {
    key: string;
    operator: TolerationOperator;
    value: string;
    taintEffect: TaintEffect;
};

export type TolerationOperator =
    | 'TOLERATION_OPERATION_UNKNOWN'
    | 'TOLERATION_OPERATOR_EXISTS'
    | 'TOLERATION_OPERATOR_EQUAL';
