export enum FetchingActionState {
    REQUEST = 'REQUEST',
    SUCCESS = 'SUCCESS',
    FAILURE = 'FAILURE',
}

export type FetchingActionTypesMap = {
    [prop in FetchingActionState]: string;
};

/**
 * Creates a map of action types with values that use the given prefix.
 *
 * @param prefix action names prefix
 * @returns map of action types for REQUEST, SUCCESS and FAILURE
 */
export function createFetchingActionTypes<T extends string>(prefix: T): FetchingActionTypesMap {
    return {
        REQUEST: `${prefix}_${FetchingActionState.REQUEST}`,
        SUCCESS: `${prefix}_${FetchingActionState.SUCCESS}`,
        FAILURE: `${prefix}_${FetchingActionState.FAILURE}`,
    };
}

export type ActionTypeInfo = {
    prefix: string;
    fetchingState: FetchingActionState;
};

/**
 * Extracts info about the action type generated with `createFetchingActionTypes`.
 *
 * @param type action type
 * @returns if the passed action type is a fetching action then returns prefix used to for that action type and
 *   fetching state (REQUEST, SUCCESS, FAILURE), otherwise returns `null`
 * @see createFetchingActionTypes
 */
export function getFetchingActionInfo(type: string): ActionTypeInfo | null {
    const matches = /(.*)_(REQUEST|SUCCESS|FAILURE)/.exec(type);
    if (!matches) {
        return null;
    }
    const [, prefix, fetchingState] = matches;
    return { prefix, fetchingState: FetchingActionState[fetchingState] };
}

export type PrefixedAction<Prefix extends string, Response> =
    | {
          type: `${Prefix}_FAILURE`;
          error: Error;
      }
    | {
          type: `${Prefix}_REQUEST`;
      }
    | {
          type: `${Prefix}_SUCCESS`;
          response: Response;
      };

export type FetchingAction<T extends Record<string, unknown>> = { type: string } & {
    [prop in keyof T]: T[prop];
};

// Action creator function types

export type RequestAction = <P>(params?: P) => FetchingAction<{ params: P | undefined }>;
export type SuccessAction = <R, P>(
    response?: R,
    params?: P
) => FetchingAction<{ response: R | undefined; params: P | undefined }>;
export type FailureAction = <P>(
    error: Error,
    params?: P
) => FetchingAction<{ error: Error; params: P | undefined }>;

function action<T extends Record<string, unknown>>(type: string, payload: T): FetchingAction<T> {
    return { type, ...payload };
}

export type FetchingActionsMap = {
    request: RequestAction;
    success: SuccessAction;
    failure: FailureAction;
};

/**
 * Creates a map of action creator functions for the given action types.
 *
 * @param types action types created with `createFetchingActionTypes`
 * @returns map with action creators for `request`, `success` and `failure` actions
 * @see createFetchingActionTypes
 */
export function createFetchingActions(types: FetchingActionTypesMap): FetchingActionsMap {
    return {
        request: (params) => action(types.REQUEST, { params }),
        success: (response, params) => action(types.SUCCESS, { response, params }),
        failure: (error, params) => action(types.FAILURE, { error, params }),
    };
}
