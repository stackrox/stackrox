import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getComplianceProfileResults } from 'services/ComplianceResultsService';

import PageTitle from 'Components/PageTitle';
import CoveragesToggleGroup from './CoveragesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfileChecksTable from './ProfileChecksTable';

function ProfileChecksPage() {
    const { profileName } = useParams();
    const pagination = useURLPagination(10);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: ['Compliance Check Name'],
        defaultSortOption: { field: 'Compliance Check Name', direction: 'asc' },
        onSort: () => setPage(1),
    });

    const fetchProfileChecks = useCallback(
        () => getComplianceProfileResults(profileName, sortOption, page, perPage),
        [page, perPage, profileName, sortOption]
    );
    const { data: profileChecks, loading: isLoading, error } = useRestQuery(fetchProfileChecks);

    return (
        <>
            <PageTitle title="Compliance coverage - Profile checks" />
            <CoveragesPageHeader />
            <PageSection>
                <CoveragesToggleGroup tableView="checks" />
            </PageSection>
            <PageSection variant="default">
                <PageSection variant="light" component="div">
                    <Title headingLevel="h2">Profile results</Title>
                    <Divider />
                    <ProfileChecksTable
                        isLoading={isLoading}
                        error={error}
                        profileChecksResults={profileChecks?.profileResults ?? []}
                        profileChecksResultsCount={profileChecks?.totalCount ?? 0}
                        profileName={profileName}
                        pagination={pagination}
                        getSortParams={getSortParams}
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ProfileChecksPage;
