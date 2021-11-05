import qs from 'qs';

/* Minimal format:
 * requires year-month-day and hour-minute (does not exclude some invalid month-day combinations)
 * does not require seconds or thousandths
 * does require UTC as time zone
 */
export const startingTimeRegExp =
    /^20\d\d-(?:0\d|1[012])-(?:0[123456789]|1\d|2\d|3[01])T(?:0\d|1\d|2[0123]):[012345]\d(?::\d\d(?:\.\d\d\d)?)?Z$/;

type QueryStringProps = {
    selectedClusterNames: string[];
    startingTimeObject: Date | null;
    isStartingTimeValid: boolean;
};

export const getQueryString = ({
    selectedClusterNames,
    startingTimeObject,
    isStartingTimeValid,
}: QueryStringProps): string => {
    // The qs package ignores params which have undefined as value.
    const queryParams = {
        cluster: selectedClusterNames.length ? selectedClusterNames : undefined,
        since:
            startingTimeObject && isStartingTimeValid
                ? startingTimeObject.toISOString()
                : undefined,
    };

    return qs.stringify(queryParams, {
        addQueryPrefix: true, // except if empty string because all params are undefined
        arrayFormat: 'repeat', // for example, cluster=abbot&cluster=costello
        encodeValuesOnly: true,
    });
};
