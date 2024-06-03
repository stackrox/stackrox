import React, { useCallback, useContext } from 'react';
import { useParams } from 'react-router-dom';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getComplianceProfileResults } from 'services/ComplianceResultsService';

import { CHECK_NAME_QUERY } from './compliance.coverage.constants';
import { DEFAULT_COMPLIANCE_PAGE_SIZE } from '../compliance.constants';
import { coverageProfileChecksPath } from './compliance.coverage.routes';
import { ComplianceProfilesContext } from './ComplianceProfilesProvider';
import ProfilesToggleGroup from './ProfilesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfileChecksTable from './ProfileChecksTable';

function ProfileChecksPage() {
    const { profileName } = useParams();
    const { profileScanStats } = useContext(ComplianceProfilesContext);
    const pagination = useURLPagination(DEFAULT_COMPLIANCE_PAGE_SIZE);
    const { page, perPage, setPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: [CHECK_NAME_QUERY],
        defaultSortOption: { field: CHECK_NAME_QUERY, direction: 'asc' },
        onSort: () => setPage(1),
    });

    const fetchProfileChecks = useCallback(
        () => getComplianceProfileResults(profileName, { sortOption, page, perPage }),
        [page, perPage, profileName, sortOption]
    );
    const { data: profileChecks, loading: isLoading, error } = useRestQuery(fetchProfileChecks);

    return (
        <>
            <PageTitle title="Compliance coverage - Profile checks" />
            <CoveragesPageHeader />
            <PageSection>
                <ProfilesToggleGroup
                    profiles={profileScanStats.scanStats}
                    route={coverageProfileChecksPath}
                />
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
