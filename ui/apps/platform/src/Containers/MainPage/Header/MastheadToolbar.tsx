import React, { ReactElement, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import {
    ApplicationLauncher,
    ApplicationLauncherGroup,
    ApplicationLauncherItem,
    PageHeaderTools,
    PageHeaderToolsGroup,
    PageHeaderToolsItem,
    ApplicationLauncherSeparator,
} from '@patternfly/react-core';
import { QuestionCircleIcon } from '@patternfly/react-icons';

import ClusterStatusProblems from 'Components/ClusterStatusProblems';
import GlobalSearchButton from 'Components/GlobalSearchButton';
import ThemeToggleButton from 'Components/ThemeToggleButton';
import useCases from 'constants/useCaseTypes';
import useMetadata from 'hooks/useMetadata';
import parseURL from 'utils/URLParser';

import CLIDownloadMenu from './CLIDownloadMenu';
import OrchestratorComponentsToggle from './OrchestratorComponentsToggle';
import UserMenu from './UserMenu';

function MastheadToolbar(): ReactElement {
    const [isHelpOpen, setIsHelpOpen] = useState(false);
    const metadata = useMetadata();
    const location = useLocation();
    const workflowState = parseURL(location);
    const useCase = workflowState.getUseCase();
    const showOrchestratorComponentsToggle =
        useCase === useCases.RISK || useCase === useCases.NETWORK;

    function toggleHelpMenu() {
        setIsHelpOpen(!isHelpOpen);
    }

    const appLauncherItems = [
        <ApplicationLauncherGroup key="app-launder-group-links">
            <ApplicationLauncherItem
                key="app-launcher-item-api"
                component={
                    <Link className="pf-c-app-launcher__menu-item" to="/main/apidocs">
                        API Reference
                    </Link>
                }
            />
            <ApplicationLauncherItem
                key="app-launcher-item-docs"
                href="/docs/product"
                isExternal
                target="_blank"
                rel="noopener noreferrer"
            >
                Help Center
            </ApplicationLauncherItem>
            <ApplicationLauncherSeparator key="separator" />
        </ApplicationLauncherGroup>,
        <ApplicationLauncherGroup key="app-launder-group-metadata">
            <ApplicationLauncherItem key="app-launcher-item-version" isDisabled>
                <span>{metadata.versionString}</span>
            </ApplicationLauncherItem>
        </ApplicationLauncherGroup>,
    ];

    return (
        <PageHeaderTools>
            {/* TODO: (PatternFly) need more robust mobile experience  than just hiding tools */}
            <PageHeaderToolsGroup visibility={{ default: 'hidden', md: 'visible' }}>
                {showOrchestratorComponentsToggle && (
                    <PageHeaderToolsItem>
                        <OrchestratorComponentsToggle useCase={useCase} />
                    </PageHeaderToolsItem>
                )}
                <PageHeaderToolsItem>
                    <GlobalSearchButton />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <CLIDownloadMenu />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <ThemeToggleButton />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <ClusterStatusProblems />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <ApplicationLauncher
                        key="help-menu"
                        aria-label="Help Menu"
                        className="co-app-launcher"
                        onToggle={toggleHelpMenu}
                        isOpen={isHelpOpen}
                        items={appLauncherItems}
                        position="right"
                        data-quickstart-id="qs-masthead-utilitymenu"
                        toggleIcon={<QuestionCircleIcon alt="" />}
                        isGrouped
                    />
                </PageHeaderToolsItem>
                <PageHeaderToolsItem>
                    <UserMenu />
                </PageHeaderToolsItem>
            </PageHeaderToolsGroup>
        </PageHeaderTools>
    );
}

export default MastheadToolbar;
