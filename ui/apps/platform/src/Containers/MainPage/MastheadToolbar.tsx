/* eslint-disable @typescript-eslint/no-unused-vars */
/* eslint-disable react/jsx-no-bind */
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
import OrchestratorComponentsToggle from 'Containers/Navigation/OrchestratorComponentsToggle';
import UserMenu from 'Containers/Navigation/UserMenu';
import useMetadata from 'hooks/useMetadata';
import parseURL from 'utils/URLParser';
import CLIDownloadMenu from './CLIDownloadMenu';

const topNavBtnClass =
    'flex flex-end px-4 no-underline pt-3 pb-2 text-base-600 items-center cursor-pointer';

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
    const appLauncherItems: ReactElement[] = [];
    appLauncherItems.push(
        <>
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
                    component={
                        <Link
                            className="pf-c-app-launcher__menu-item"
                            target="_blank"
                            rel="noopener noreferrer"
                            to="/docs/product"
                        >
                            Help Center
                        </Link>
                    }
                />
                <ApplicationLauncherSeparator key="separator" />
            </ApplicationLauncherGroup>
            <ApplicationLauncherGroup key="app-launder-group-metadata">
                <ApplicationLauncherItem key="app-launcher-item-version" isDisabled>
                    <span>{metadata.versionString}</span>
                </ApplicationLauncherItem>
            </ApplicationLauncherGroup>
        </>
    );

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
                    <GlobalSearchButton topNavBtnClass={topNavBtnClass} />
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
                        onSelect={() => {}}
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
