import { useEffect } from 'react';

import useURLParameter, { Action } from 'hooks/useURLParameter';
import { NonEmptyArray } from 'utils/type.utils';

export type UseURLStringUnionReturn<Value> = [
    Value,
    (nextValue: unknown, historyAction?: Action) => void,
];

function isValidValue<Values extends Readonly<NonEmptyArray<unknown>>>(
    values: Values,
    value: unknown
): value is Values[number] {
    return values.some((v) => v === value);
}

/**
 * Hook that provides a type-safe way to read/write a set of string values to a URL parameter. The
 * setter function returned by this hook can be called with a type wider than the string union, but
 * will only set the parameter if the passed value is valid. Invalid values passed to the setter
 * are a no-op.
 *
 * @param parameterName The name of the URL parameter key
 * @param values A non-empty tuple of values that are valid for the parameter
 * @param defaultValue The default value in the URL, defaults to the first item of `values`
 *
 * @returns A tuple of length two containing the current value, and a function to set the value
 */
export default function useURLStringUnion<Values extends Readonly<NonEmptyArray<string>>>(
    parameterName: string,
    values: Values,
    defaultValue: Values[number] = values[0]
): UseURLStringUnionReturn<Values[number]> {
    const [paramValue, setParamValue] = useURLParameter(parameterName, defaultValue);
    // Ensures an incorrect value entered into the URL is replaced with the default value
    const currentValue = isValidValue(values, paramValue) ? paramValue : defaultValue;

    // Synchronizes the URL with the default parameter value on the initial hook call
    useEffect(() => {
        setParamValue(currentValue, 'replace');
    }, [currentValue, setParamValue]);

    function safeSetValue(nextValue: unknown, historyAction?: Action) {
        // Ensures the value cannot be set incorrectly by calling code
        if (isValidValue(values, nextValue)) {
            setParamValue(nextValue, historyAction);
        }
    }

    return [currentValue, safeSetValue];
}
