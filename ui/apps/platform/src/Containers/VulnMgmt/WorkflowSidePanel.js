import React from 'react';
import { withRouter, Link } from 'react-router-dom';
import { ExternalLink } from 'react-feather';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';

import EntityBreadCrumbs from './EntityBreadCrumbs';

const WorkflowSidePanel = ({ history, location, children }) => {
    const workflowState = parseURL(location);
    const pageStack = workflowState.getPageStack();
    const breadCrumbEntities = workflowState.stateStack.slice(pageStack.length);

    function onClose() {
        const url = workflowState.removeSidePanelParams().toUrl();
        history.push(url);
    }

    const url = workflowState.getSkimmedStack().toUrl();
    const externalLink = (
        <div className="flex items-center h-full hover:bg-base-300">
            <Link
                to={url}
                aria-label="External link"
                className="border-base-400 border-l h-full p-4"
            >
                <ExternalLink className="h-6 w-6 text-base-600" />
            </Link>
        </div>
    );

    return (
        <workflowStateContext.Provider value={workflowState}>
            <PanelNew testid="side-panel">
                <PanelHead>
                    <EntityBreadCrumbs workflowEntities={breadCrumbEntities} />
                    <PanelHeadEnd>
                        {externalLink}
                        <CloseButton onClose={onClose} className="border-base-400 border-l" />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>{children}</PanelBody>
            </PanelNew>
        </workflowStateContext.Provider>
    );
};

/*
 * If more than one SidePanel is rendered, this Pure Functional Component will need to be converted to
 * a Class Component in order to work correctly. See https://github.com/stackrox/rox/pull/3090#pullrequestreview-274948849
 */
export default withRouter(WorkflowSidePanel);
