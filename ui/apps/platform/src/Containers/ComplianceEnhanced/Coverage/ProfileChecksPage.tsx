import React, { useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import { getComplianceProfileResults } from 'services/ComplianceResultsService';

import PageTitle from 'Components/PageTitle';
import CoveragesToggleGroup from './CoveragesToggleGroup';
import CoveragesPageHeader from './CoveragesPageHeader';
import ProfileChecksTable from './ProfileChecksTable';

function ProfileChecksPage() {
    const { profileName } = useParams();

    const fetchProfileChecks = useCallback(
        () => getComplianceProfileResults(profileName),
        [profileName]
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
                        profileChecks={profileChecks?.profileResults ?? []}
                        profileName={profileName}
                    />
                </PageSection>
            </PageSection>
        </>
    );
}

export default ProfileChecksPage;
