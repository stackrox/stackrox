import { useEffect, useState } from 'react';
import { fetchAccessScopes } from 'services/AccessScopesService';
import { getCollection } from 'services/CollectionsService';

import { fetchReportById } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';
import useFeatureFlags from './useFeatureFlags';

export type ReportScope = {
    // The 'AccessControlScope' type is deprecated and should be able to be removed in the release after 3.74
    type: 'CollectionScope' | 'AccessControlScope';
    id: string;
    name: string;
    description: string;
};

type Result = {
    isLoading: boolean;
    report: ReportConfiguration | null;
    reportScope: ReportScope | null;
    error: string | null;
};

const defaultResultState = {
    report: null,
    reportScope: null,
    error: null,
    isLoading: true,
};

function fetchCollectionReportScope(scopeId: string): Promise<ReportScope> {
    const { request } = getCollection(scopeId);

    return request.then(({ collection: { id, name, description } }) => ({
        type: 'CollectionScope',
        id,
        name,
        description,
    }));
}

function fetchAccessScopeReportScope(scopeId: string): Promise<ReportScope | null> {
    return fetchAccessScopes().then((scopes) => {
        const fullScope = scopes.find((scope) => scope.id === scopeId);

        if (!fullScope) {
            return null;
        }

        const { id, name, description } = fullScope;
        return { type: 'AccessControlScope', id, name, description };
    });
}

/*
    When migrating the report config from using access scopes as the "report scope", to using
    collections, there are some access scope configurations that could not be converted. In this
    case, the `scopeId` attached to the report will still reference an access scope. If our request
    to obtain the collection fails (since the ID is not for a collection), we make a second request
    for an access scope using the same ID. If this request succeeds, we know that the user has an
    invalid access scope attached to the report and we can use this information to prompt them
    to configure a collection.

    TODO - In the next release when the feature flag is removed:
    This impacts systems that have upgraded from 3.73 to 3.74. In the release after 3.74 we can 
    remove this check, as all invalid report configurations will be removed.
*/
function fetchReportScope(scopeId: string, isCollectionsEnabled: boolean) {
    if (isCollectionsEnabled) {
        return fetchCollectionReportScope(scopeId).catch(() =>
            // Could not get a collection, so try to get an access scope instead
            fetchAccessScopeReportScope(scopeId)
        );
    }
    return fetchAccessScopeReportScope(scopeId);
}

/*
 * This hook does an API call to the report configurations API to get the list of reports
 */
function useFetchReport(reportId: string, refresh = 0): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isCollectionsEnabled = isFeatureFlagEnabled('ROX_POSTGRES_DATASTORE');

    useEffect(() => {
        setResult(defaultResultState);

        if (reportId) {
            fetchReportById(reportId)
                .then((report) =>
                    fetchReportScope(report.scopeId, isCollectionsEnabled).then((reportScope) => {
                        setResult({ report, reportScope, error: null, isLoading: false });
                    })
                )
                .catch((error) => {
                    setResult({ report: null, error, isLoading: false, reportScope: null });
                });
        }
    }, [reportId, refresh, isCollectionsEnabled]);

    return result;
}

export default useFetchReport;
