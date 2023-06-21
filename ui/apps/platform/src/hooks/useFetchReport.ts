import { useEffect, useState } from 'react';
import { getCollection } from 'services/CollectionsService';

import { fetchReportById } from 'services/ReportsService';
import { ReportConfiguration } from 'types/report.proto';

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

/*
 * This hook does an API call to the report configurations API to get the list of reports
 */
function useFetchReport(reportId: string, refresh = 0): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        if (reportId) {
            fetchReportById(reportId)
                .then((report) =>
                    fetchCollectionReportScope(report.scopeId).then((reportScope) => {
                        setResult({ report, reportScope, error: null, isLoading: false });
                    })
                )
                .catch((error) => {
                    setResult({ report: null, error, isLoading: false, reportScope: null });
                });
        }
    }, [reportId, refresh]);

    return result;
}

export default useFetchReport;
