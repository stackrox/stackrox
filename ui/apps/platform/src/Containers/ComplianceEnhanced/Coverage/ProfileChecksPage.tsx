import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import { getComplianceProfileResults } from 'services/ComplianceResultsService';

import PageTitle from 'Components/PageTitle';
import CoveragesToggleGroup from './CoveragesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfileChecksTable from './ProfileChecksTable';

function ProfileChecksPage() {
    const { profileName } = useParams();
    const pagination = useURLPagination(10);

    const { page, perPage } = pagination;

    const fetchProfileChecks = useCallback(
        () => getComplianceProfileResults(profileName, page, perPage),
        [page, perPage, profileName]
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
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ProfileChecksPage;
