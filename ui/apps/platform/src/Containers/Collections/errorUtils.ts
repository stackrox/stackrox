import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

type CollectionConfigErrorType =
    | 'CollectionLoop'
    | 'DuplicateName'
    | 'EmptyName'
    | 'InvalidRule'
    | 'EmptyCollection'
    | 'UnknownError';

export type CollectionConfigError =
    | {
          type: 'CollectionLoop';
          message: string;
          details?: string;
          loopId: string | undefined;
      }
    | {
          type: Exclude<CollectionConfigErrorType, 'CollectionLoop'>;
          message: string;
          details?: string;
      };

/**
 * Given an error, attempt to categorize and provide more information to the user. These are
 * errors that cannot be detected by general `yup` validation and often can only be detected
 * by the server.
 *
 * @param err An instance of an Error
 * @return A categorized error specific to collections
 */
export function parseConfigError(err: Error): CollectionConfigError {
    const rawMessage = getAxiosErrorMessage(err);

    if (/create a loop/.test(rawMessage)) {
        const errorRegex = /^edge between '[0-9a-fA-F-]*' and '(?<loopId>[0-9a-fA-F-]*)'/;
        const matches = errorRegex.exec(rawMessage);
        const loopId = matches?.groups?.loopId;
        return {
            type: 'CollectionLoop',
            message: 'An attached collection has created a loop, which is not supported',
            details: 'Detach the invalid collection',
            loopId,
        };
    }

    if (
        // Error for collection save
        /collections must have non-empty, unique `name` values/.test(rawMessage) ||
        // Error for collection update
        /name already in use/.test(rawMessage)
    ) {
        return { type: 'DuplicateName', message: 'The collection name must be unique' };
    }

    if (/name should not be empty/.test(rawMessage)) {
        return { type: 'EmptyName', message: 'A name value is required for a collection' };
    }

    if (/failed to compile regex/.test(rawMessage)) {
        return {
            type: 'InvalidRule',
            message:
                'The server was unable to process a regular expression used in a collection rule',
            details: rawMessage,
        };
    }

    if (/Cannot save an empty collection/.test(rawMessage)) {
        return {
            type: 'EmptyCollection',
            message: rawMessage,
        };
    }

    return {
        type: 'UnknownError',
        message: 'An unexpected error has occurred when processing the collection',
        details: rawMessage,
    };
}
