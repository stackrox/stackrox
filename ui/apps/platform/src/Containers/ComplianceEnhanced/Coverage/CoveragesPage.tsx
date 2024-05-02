import React, { useCallback, useEffect } from 'react';
import { Route, Switch, useHistory, useParams } from 'react-router-dom';
import { Bullseye, PageSection, Spinner, Text, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import { complianceEnhancedCoveragePath } from 'routePaths';
import { getComplianceProfilesStats } from 'services/ComplianceResultsService';

import ProfileChecksPage from './ProfileChecksPage';
import ProfileClustersPage from './ProfileClustersPage';

function CoveragesPage() {
    const fetchProfilesStats = useCallback(() => getComplianceProfilesStats(), []);
    const { data: profileScanStats, loading: isLoading, error } = useRestQuery(fetchProfilesStats);

    const history = useHistory();
    const { profileName } = useParams();

    const { scanStats } = profileScanStats || {};

    // redirect to the first profile if no profile is given
    useEffect(() => {
        if (scanStats && scanStats.length > 0 && !profileName) {
            const firstProfileName = scanStats[0].profileName;
            history.push(`${complianceEnhancedCoveragePath}/profiles/${firstProfileName}/checks`);
        }
    }, [isLoading, scanStats, profileName, history]);

    if (isLoading) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return <div>Error: {error.message}</div>;
    }

    if (!profileScanStats) {
        return <div>No results</div>;
    }

    return (
        <>
            <PageTitle title="Compliance coverage" />
            <PageSection component="div" variant="light">
                <Title headingLevel="h1">Compliance coverage</Title>
                <Text>
                    Assess profile compliance for nodes and platform resources across clusters
                </Text>
            </PageSection>
            <PageSection>
                <Switch>
                    <Route
                        exact
                        path={`${complianceEnhancedCoveragePath}/profiles/:profileName/checks`}
                        render={() => <ProfileChecksPage profileScanStats={profileScanStats} />}
                    />
                    <Route
                        exact
                        path={`${complianceEnhancedCoveragePath}/profiles/:profileName/clusters`}
                        render={() => <ProfileClustersPage profileScanStats={profileScanStats} />}
                    />
                </Switch>
            </PageSection>
        </>
    );
}

export default CoveragesPage;
