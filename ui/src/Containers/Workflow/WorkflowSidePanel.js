import React from 'react';
import { withRouter, Link } from 'react-router-dom';
import parseURL from 'modules/URLParser';
import onClickOutside from 'react-onclickoutside';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import { ExternalLink as ExternalLinkIcon } from 'react-feather';
import Panel from 'Components/Panel';
import SidePanelAnimation from 'Components/animations/SidePanelAnimation';
import EntityBreadCrumbs from 'Containers/BreadCrumbs/EntityBreadCrumbs';

const WorkflowSidePanel = ({ history, location, children, isOpen }) => {
    const { isDarkMode } = useTheme();
    const workflowState = parseURL(location);
    const pageStack = workflowState.getPageStack();
    const breadCrumbEntities = workflowState.stateStack.slice(pageStack.length);

    const firstItem = workflowState.getBaseEntity();
    const isList = firstItem.entityType && !firstItem.entityId;

    function onClose() {
        const url = workflowState.removeSidePanelParams().toUrl();
        history.push(url);
    }

    WorkflowSidePanel.handleClickOutside = () => {
        onClose();
    };

    const url = workflowState.skimStack().toUrl();
    const externalLink = (
        <div className="flex items-center h-full hover:bg-base-300">
            <Link
                to={url}
                data-test-id="external-link"
                className={`${
                    !isDarkMode ? 'border-base-100' : 'border-base-400'
                } border-l h-full p-4`}
            >
                <ExternalLinkIcon className="h-6 w-6 text-base-600" />
            </Link>
        </div>
    );

    return (
        <workflowStateContext.Provider value={workflowState}>
            <SidePanelAnimation condition={isOpen}>
                <div className="w-full h-full bg-base-100 border-l border-base-400 shadow-sidepanel">
                    <Panel
                        id="side-panel"
                        headerClassName={`flex w-full h-14 overflow-y-hidden border-b ${
                            !isDarkMode ? 'bg-side-panel-wave border-base-100' : 'border-base-400'
                        }`}
                        bodyClassName={`${isList || isDarkMode ? 'bg-base-100' : ''}`}
                        headerTextComponent={
                            <EntityBreadCrumbs workflowEntities={breadCrumbEntities} />
                        }
                        headerComponents={externalLink}
                        onClose={onClose}
                        closeButtonClassName={
                            isDarkMode ? 'border-l border-base-400' : 'border-l border-base-100'
                        }
                    >
                        {children}
                    </Panel>
                </div>
            </SidePanelAnimation>
        </workflowStateContext.Provider>
    );
};

const clickOutsideConfig = {
    handleClickOutside: () => () => WorkflowSidePanel.handleClickOutside()
};

/*
 * If more than one SidePanel is rendered, this Pure Functional Component will need to be converted to
 * a Class Component in order to work correctly. See https://github.com/stackrox/rox/pull/3090#pullrequestreview-274948849
 */
export default onClickOutside(withRouter(WorkflowSidePanel), clickOutsideConfig);
