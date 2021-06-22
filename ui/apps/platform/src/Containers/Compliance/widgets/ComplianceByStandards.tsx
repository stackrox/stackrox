import React, { ReactElement } from 'react';
import { useQuery } from '@apollo/client';

import { ResourceType } from 'constants/entityTypes';
import Loader from 'Components/Loader';
import { STANDARDS_QUERY } from 'queries/standard';
import ComplianceByStandard from './ComplianceByStandard';

type ComplianceByStandardsProps = {
    entityType?: ResourceType;
};

function ComplianceByStandards({ entityType }: ComplianceByStandardsProps): ReactElement {
    const { loading, data } = useQuery(STANDARDS_QUERY);
    if (loading) {
        return <Loader />;
    }
    let standards = data.results;
    if (entityType) {
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
                    className="pdf-page"
                />
            ))}
        </>
    );
}

export default ComplianceByStandards;
