import React from 'react';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { Link, useNavigate, useLocation } from 'react-router-dom';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
import workflowStateContext from 'Containers/workflowStateContext';
import parseURL from 'utils/URLParser';

import EntityBreadCrumbs from './EntityBreadCrumbs';

const WorkflowSidePanel = ({ children }) => {
    const navigate = useNavigate();
    const location = useLocation();
    const workflowState = parseURL(location);
    const pageStack = workflowState.getPageStack();
    const breadCrumbEntities = workflowState.stateStack.slice(pageStack.length);

    function onClose() {
        const url = workflowState.removeSidePanelParams().toUrl();
        navigate(url);
    }

    const url = workflowState.getSkimmedStack().toUrl();
    const externalLink = (
        <div className="flex items-center h-full hover:bg-base-300">
            <Link to={url} aria-label="link" className="border-base-400 border-l h-full p-4">
                <ExternalLinkAltIcon />
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
export default WorkflowSidePanel;
