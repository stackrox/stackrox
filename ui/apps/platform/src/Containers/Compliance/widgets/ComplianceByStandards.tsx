import React, { ReactElement } from 'react';
import { useQuery } from '@apollo/client';

import Loader from 'Components/Loader';
import { STANDARDS_QUERY } from 'queries/standard';
import { ComplianceStandardScope } from 'services/ComplianceService';

import ComplianceByStandard from './ComplianceByStandard';

type StandardsQueryDataType = {
    results: {
        id: string;
        name: string;
        scopes: ComplianceStandardScope[];
        hidden: boolean;
    }[];
};

type ComplianceByStandardsProps = {
    entityId: string;
    entityName: string;
    entityType: ComplianceStandardScope;
};

function ComplianceByStandards({
    entityId,
    entityName,
    entityType,
}: ComplianceByStandardsProps): ReactElement {
    const { loading, data, error } = useQuery<StandardsQueryDataType>(STANDARDS_QUERY);
    if (loading) {
        return <Loader />;
    }

    if (error) {
        return (
            <div>
                A database error has occurred. Please check that you have the correct permissions to
                view this information.
            </div>
        );
    }

    /* eslint-disable no-nested-ternary */
    const standards = !data?.results
        ? []
        : !entityType
        ? data.results
        : data.results.filter(({ scopes }) => scopes.includes(entityType));
    /* eslint-enable no-nested-ternary */

    return (
        <>
            {standards.map(({ name: standardName, id: standardId }) => (
                <ComplianceByStandard
                    key={standardId}
                    standardName={standardName}
                    standardId={standardId}
                    entityId={entityId}
                    entityName={entityName}
                    entityType={entityType}
                    className="pdf-page"
                />
            ))}
        </>
    );
}

export default ComplianceByStandards;
