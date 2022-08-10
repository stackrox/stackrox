import React from 'react';

import { Divider, PageSection, Title } from '@patternfly/react-core';
import pluralize from 'pluralize';

import PluginProvider from 'console-plugins/PluginProvider';
import ViolationsByPolicyCategory from 'Containers/Dashboard/PatternFly/Widgets/ViolationsByPolicyCategory';
import PolicyViolationTiles from 'Containers/Dashboard/PatternFly/Widgets/PolicyViolationTiles';
import useAlertGroups from 'Containers/Dashboard/PatternFly/hooks/useAlertGroups';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

export default function Overview() {
    const countsFilterQuery = getRequestQueryStringForSearchFilter({});
    const { data } = useAlertGroups(countsFilterQuery);

    const severityCounts = (data && data[0]?.counts) ?? [];

    const counts = {
        LOW_SEVERITY: 0,
        MEDIUM_SEVERITY: 0,
        HIGH_SEVERITY: 0,
        CRITICAL_SEVERITY: 0,
    };
    let totalCount = 0;
    severityCounts.forEach(({ severity, count }) => {
        counts[severity] = parseInt(count, 10);
        totalCount += counts[severity];
    });

    return (
        <PluginProvider>
            <PageSection>
                <Title headingLevel="h1">Dashboard</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection className="pf-u-m-lg">
                {
                    // Any dashboard components using react-table blow up in production... not sure why
                    /* <ViolationsByPolicySeverity /> */
                }
                <Title headingLevel="h2" className="pf-u-pb-md">
                    {`${totalCount} policy ${pluralize('violation', totalCount)} by severity`}
                </Title>
                <PolicyViolationTiles searchFilter={{}} counts={counts} />
                <Divider className="pf-u-my-lg" component="div" />
                <ViolationsByPolicyCategory />
            </PageSection>
        </PluginProvider>
    );
}
