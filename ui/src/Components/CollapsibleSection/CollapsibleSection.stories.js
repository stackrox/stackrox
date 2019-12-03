import React from 'react';
import { Plus, RefreshCw } from 'react-feather';

import PanelButton from 'Components/PanelButton';

import CollapsibleSection from './CollapsibleSection';

export default {
    title: 'CollapsibleSection',
    component: CollapsibleSection
};

export const withTitleOnly = () => {
    return (
        <CollapsibleSection title="Policy summary">
            <div className="mb-4 pdf">
                <h2>Content of collapsible section</h2>
                <p>
                    Try a more powerful colour you can get my logo from facebook but can we try some
                    other colours maybe. It looks a bit empty, try to make everything bigger make it
                    pop.
                </p>
            </div>
        </CollapsibleSection>
    );
};

export const withActionButtons = () => {
    function dummyFunc(event) {
        // eslint-disable-next-line no-alert
        alert(`${event.target.textContent} clicked`);
    }

    const headerComponents = (
        <>
            <PanelButton
                icon={<RefreshCw className="h-4 w-4 ml-1" />}
                className="btn btn-base mr-2"
                onClick={dummyFunc}
                tooltip="Manually enrich external data"
                disabled={false}
            >
                Reassess All
            </PanelButton>
            <PanelButton
                icon={<Plus className="h-4 w-4 ml-1" />}
                className="btn btn-base"
                onClick={dummyFunc}
                disabled={false}
            >
                New Policy
            </PanelButton>
        </>
    );

    return (
        <CollapsibleSection title="Policy summary" headerComponents={headerComponents}>
            <div className="mb-4 pdf">
                <h2>Content of collapsible section</h2>
                <p>
                    Try a more powerful colour you can get my logo from facebook but can we try some
                    other colours maybe. It looks a bit empty, try to make everything bigger make it
                    pop.
                </p>
            </div>
        </CollapsibleSection>
    );
};
