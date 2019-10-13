import React, { useContext } from 'react';
import { withRouter, Link } from 'react-router-dom';
import WorkflowStateMgr from 'modules/WorkflowStateManager';
import { generateURL } from 'modules/URLReadWrite';
import onClickOutside from 'react-onclickoutside';
import { useTheme } from 'Containers/ThemeProvider';
import workflowStateContext from 'Containers/workflowStateContext';
import { ExternalLink as ExternalLinkIcon } from 'react-feather';
import Panel from 'Components/Panel';
import SidePanelAnimation from 'Components/animations/SidePanelAnimation';
import searchContexts from 'constants/searchContexts';
import searchContext from 'Containers/searchContext';

const WorkflowSidePanel = ({ history, children, isOpen }) => {
    const { isDarkMode } = useTheme();
    const workflowState = useContext(workflowStateContext);
    const { useCase } = workflowState;
    const firstItem = workflowState.getBaseEntity();
    const currentItem = workflowState.getCurrentEntity();

    const isList = firstItem.entityType && !firstItem.entityId;

    function onClose() {
        const workflowStateMgr = new WorkflowStateMgr(workflowState);
        workflowStateMgr.removeSidePanelParams();
        const url = generateURL(workflowStateMgr.workflowState);
        history.push(url);
    }

    WorkflowSidePanel.handleClickOutside = () => {
        onClose();
    };

    // const { loading, error, data } = useQuery(query, { variables: { id: currentItem.entityId } });

    const workflowStateMgr = new WorkflowStateMgr(workflowState);
    workflowStateMgr.reset(useCase, currentItem.entityType, currentItem.entityId);
    const url = generateURL(workflowStateMgr.workflowState);
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
        <searchContext.Provider value={searchContexts.sidePanel}>
            <SidePanelAnimation condition={isOpen}>
                <div className="w-full h-full bg-base-100 border-l border-base-400 shadow-sidepanel">
                    <Panel
                        id="side-panel"
                        headerClassName={`flex w-full h-14 overflow-y-hidden border-b ${
                            !isDarkMode ? 'bg-side-panel-wave border-base-100' : 'border-base-400'
                        }`}
                        bodyClassName={`${isList || isDarkMode ? 'bg-base-100' : ''}`}
                        headerTextComponent={
                            <div>TODO: Breadcrumbs</div>
                            // <BreadCrumbs
                            //     className="font-700 leading-normal text-base-600 tracking-wide truncate"
                            //     entityType1={entityType1 || entityListType1}
                            //     entityId1={entityId1}
                            //     entityType2={entityType2}
                            //     entityListType2={entityListType2}
                            //     entityId2={entityId2}
                            // />
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
        </searchContext.Provider>
    );
};

const clickOutsideConfig = {
    handleClickOutside: () => WorkflowSidePanel.handleClickOutside
};

/*
 * If more than one SidePanel is rendered, this Pure Functional Component will need to be converted to
 * a Class Component in order to work correctly. See https://github.com/stackrox/rox/pull/3090#pullrequestreview-274948849
 */
export default onClickOutside(withRouter(WorkflowSidePanel), clickOutsideConfig);
