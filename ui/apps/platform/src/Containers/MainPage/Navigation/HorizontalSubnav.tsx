import { Suspense, lazy } from 'react';
import type { ComponentType, ReactElement } from 'react';
import { matchPath, useLocation } from 'react-router-dom-v5-compat';
import { Nav } from '@patternfly/react-core';

import {
    everyResource,
    someResource,
    violationsBasePath,
    vulnerabilitiesAllImagesPath,
    vulnerabilitiesImagesWithoutCvesPath,
    vulnerabilitiesInactiveImagesPath,
    vulnerabilitiesNodeCvesPath,
    vulnerabilitiesPlatformCvesPath,
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesVirtualMachineCvesPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import type { IsFeatureFlagEnabled } from 'hooks/useFeatureFlags';
import type { HasReadAccess } from 'hooks/usePermissions';
import type { FeatureFlagPredicate } from 'utils/featureFlagUtils';

import './HorizontalSubnav.css';

const ViolationsSubnav = lazy(() => import('Containers/Violations/ViolationsSubnav'));
const VulnerabilitiesSubnav = lazy(
    () => import('Containers/Vulnerabilities/VulnerabilitiesSubnav')
);

type SubnavRouteKey = 'violations' | 'vulnerabilities';

type SubnavRouteConfig = {
    key: SubnavRouteKey;
    patterns: string[];
    component: ComponentType<{
        hasReadAccess: HasReadAccess;
        isFeatureFlagEnabled: IsFeatureFlagEnabled;
    }>;
    featureFlagRequirements?: FeatureFlagPredicate;
    resourceAccessRequirements?: (hasReadAccess: HasReadAccess) => boolean;
};

const subnavRoutes: SubnavRouteConfig[] = [
    {
        key: 'violations',
        patterns: [`${violationsBasePath}/*`],
        component: ViolationsSubnav,
        resourceAccessRequirements: everyResource(['Alert']),
    },
    {
        key: 'vulnerabilities',
        patterns: [
            `${vulnerabilitiesUserWorkloadsPath}/*`,
            `${vulnerabilitiesPlatformPath}/*`,
            `${vulnerabilitiesNodeCvesPath}/*`,
            `${vulnerabilitiesVirtualMachineCvesPath}/*`,
            `${vulnerabilitiesAllImagesPath}/*`,
            `${vulnerabilitiesInactiveImagesPath}/*`,
            `${vulnerabilitiesImagesWithoutCvesPath}/*`,
            `${vulnerabilitiesPlatformCvesPath}/*`,
            `${vulnerabilitiesWorkloadCvesPath}/*`, // Legacy (TODO: deprecate)
        ],
        component: VulnerabilitiesSubnav,
        // Requires any of the core vulnerability resources - individual routes filtered in subnav component
        resourceAccessRequirements: someResource(['Deployment', 'Image', 'Cluster', 'Node']),
    },
];

function getSubnavComponentForPath(
    pathname: string,
    hasReadAccess: HasReadAccess,
    isFeatureFlagEnabled: IsFeatureFlagEnabled
): ComponentType<{
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
}> | null {
    const route = subnavRoutes.find(({ patterns }) =>
        patterns.some((path) => matchPath({ path }, pathname))
    );

    if (!route) {
        return null;
    }

    const { component, featureFlagRequirements, resourceAccessRequirements } = route;
    const areFeatureFlagRequirementsMet = featureFlagRequirements?.(isFeatureFlagEnabled) ?? true;
    const areResourceAccessRequirementsMet = resourceAccessRequirements?.(hasReadAccess) ?? true;

    return areFeatureFlagRequirementsMet && areResourceAccessRequirementsMet ? component : null;
}

export type HorizontalSubnavProps = {
    hasReadAccess: HasReadAccess;
    isFeatureFlagEnabled: IsFeatureFlagEnabled;
};

function HorizontalSubnav({
    hasReadAccess,
    isFeatureFlagEnabled,
}: HorizontalSubnavProps): ReactElement | null {
    const location = useLocation();
    const SubnavComponent = getSubnavComponentForPath(
        location.pathname,
        hasReadAccess,
        isFeatureFlagEnabled
    );

    if (!SubnavComponent) {
        return null;
    }

    return (
        <Suspense fallback={null}>
            <Nav variant="horizontal-subnav" className="acs-pf-horizontal-subnav">
                <SubnavComponent
                    hasReadAccess={hasReadAccess}
                    isFeatureFlagEnabled={isFeatureFlagEnabled}
                />
            </Nav>
        </Suspense>
    );
}

export default HorizontalSubnav;
