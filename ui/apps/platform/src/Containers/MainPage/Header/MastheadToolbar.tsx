import React from 'react';
import type { ReactElement } from 'react';
import { useLocation } from 'react-router-dom-v5-compat';
import { Flex, FlexItem } from '@patternfly/react-core';

import useCases from 'constants/useCaseTypes';
import useIsRouteEnabled from 'hooks/useIsRouteEnabled';
import parseURL from 'utils/URLParser';
import { searchPath } from 'routePaths';

import CLIDownloadMenu from './CLIDownloadMenu';
import ClusterStatusProblems from './ClusterStatusProblems';
import GlobalSearchButton from './GlobalSearchButton';
import HelpMenu from './HelpMenu';
import OrchestratorComponentsToggle from './OrchestratorComponentsToggle';
/*
 * TODO: remove this comment, which hides the light-mode/dark-mode toggle import,
 *       after we update to use PatternFly themes for dark mode
import ThemeToggleButton from './ThemeToggleButton';
 */
import UserMenu from './UserMenu';

function MastheadToolbar(): ReactElement {
    const isRouteEnabled = useIsRouteEnabled();
    const isRouteEnabledForSearch = isRouteEnabled('search');

    const location = useLocation();
    const workflowState = parseURL(location);
    const useCase = workflowState.getUseCase();
    const showOrchestratorComponentsToggle =
        useCase === useCases.RISK || location.pathname === searchPath;

    // TODO: (PatternFly) need more robust mobile experience than just hiding tools
    // <PageHeaderToolsGroup visibility={{ default: 'hidden', md: 'visible' }}>
    return (
        <Flex spaceItems={{ default: 'spaceItemsSm' }}>
            {showOrchestratorComponentsToggle && (
                <FlexItem>
                    <OrchestratorComponentsToggle />
                </FlexItem>
            )}
            {isRouteEnabledForSearch && (
                <FlexItem>
                    <GlobalSearchButton />
                </FlexItem>
            )}
            <FlexItem>
                <CLIDownloadMenu />
            </FlexItem>
            {/*
                * TODO: remove this comment, which hides the light-mode/dark-mode toggle,
                *       after we update to use PatternFly themes for dark mode
            <FlexItem>
                <ThemeToggleButton />
            </FlexItem>
            */}
            <FlexItem>
                <ClusterStatusProblems />
            </FlexItem>
            <FlexItem>
                <HelpMenu />
            </FlexItem>
            <FlexItem>
                <UserMenu />
            </FlexItem>
        </Flex>
    );
}

export default MastheadToolbar;
