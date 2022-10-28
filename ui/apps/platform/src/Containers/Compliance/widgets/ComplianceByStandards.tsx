import React, { ReactElement } from 'react';
import { useQuery } from '@apollo/client';

import { ResourceType } from 'constants/entityTypes';
import Loader from 'Components/Loader';
import { STANDARDS_QUERY } from 'queries/standard';
import ComplianceByStandard from './ComplianceByStandard';

type ComplianceByStandardsProps = {
    entityId?: string;
    entityName?: string;
    entityType?: ResourceType;
};

function ComplianceByStandards({
    entityId,
    entityName,
    entityType,
}: ComplianceByStandardsProps): ReactElement {
    const { loading, data, error } = useQuery(STANDARDS_QUERY);
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

    let standards = data?.results || [];
    if (entityType && Array.isArray(data?.results)) {
        standards = data.results.filter(
            ({ scopes }): boolean => scopes.includes(entityType) as boolean
        );
    }
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
