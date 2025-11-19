import qs from 'qs';

type QueryStringProps = {
    selectedClusterNames: string[];
    startingTimeIso: string | null;
    isDatabaseDiagnosticsOnly: boolean;
    includeComplianceOperatorResources: boolean;
};

export const getQueryString = ({
    selectedClusterNames,
    startingTimeIso,
    isDatabaseDiagnosticsOnly,
    includeComplianceOperatorResources,
}: QueryStringProps): string => {
    // The qs package ignores params which have undefined as value.
    const queryParams = {
        'database-only': isDatabaseDiagnosticsOnly || undefined,
        'compliance-operator': includeComplianceOperatorResources || undefined,
        since: startingTimeIso || undefined,
        cluster: selectedClusterNames.length > 0 ? selectedClusterNames : undefined,
    };

    return qs.stringify(queryParams, {
        addQueryPrefix: true, // except if empty string because all params are undefined
        arrayFormat: 'repeat', // for example, cluster=abbot&cluster=costello
        encodeValuesOnly: true,
    });
};
