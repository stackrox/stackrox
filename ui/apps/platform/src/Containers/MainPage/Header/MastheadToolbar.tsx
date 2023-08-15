import React, { ReactElement } from 'react';
import { useLocation } from 'react-router-dom';
import { PageHeaderTools, PageHeaderToolsGroup, PageHeaderToolsItem } from '@patternfly/react-core';

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
    const isRouteEnabledForSearch = isRouteEnabled(searchPath);

    const location = useLocation();
    const workflowState = parseURL(location);
    const useCase = workflowState.getUseCase();
    const showOrchestratorComponentsToggle =
        useCase === useCases.RISK || location.pathname === searchPath;

    return (
        <PageHeaderTools>
            {/* TODO: (PatternFly) need more robust mobile experience  than just hiding tools */}
            <PageHeaderToolsGroup visibility={{ default: 'hidden', md: 'visible' }}>
                {showOrchestratorComponentsToggle && (
                    <PageHeaderToolsItem>
                        <OrchestratorComponentsToggle />
                    </PageHeaderToolsItem>
                )}
                {isRouteEnabledForSearch && (
                    <PageHeaderToolsItem>
                        <GlobalSearchButton />
                    </PageHeaderToolsItem>
                )}
                <PageHeaderToolsItem>
                    <CLIDownloadMenu />
                </PageHeaderToolsItem>
                {/*
                  * TODO: remove this comment, which hides the light-mode/dark-mode toggle,
                  *       after we update to use PatternFly themes for dark mode
                <PageHeaderToolsItem>
                    <ThemeToggleButton />
                </PageHeaderToolsItem>
                */}
                <PageHeaderToolsItem>
                    <ClusterStatusProblems />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <HelpMenu />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <UserMenu />
                </PageHeaderToolsItem>
            </PageHeaderToolsGroup>
        </PageHeaderTools>
    );
}

export default MastheadToolbar;
