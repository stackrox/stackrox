import React, { ReactElement } from 'react';

import Tabs from 'Components/Tabs';
import Tab from 'Components/Tab';
import { NetworkPolicyModification } from 'Containers/Network/networkTypes';

type SuccessViewTabsProps = {
    modificationName?: string;
    modification?: NetworkPolicyModification;
};

function SuccessViewTabs({
    modificationName = '',
    modification,
}: SuccessViewTabsProps): ReactElement {
    const tabs = [{ text: modificationName }];
    const { applyYaml, toDelete } = modification || {};
    const shouldDelete = toDelete && toDelete.length > 0;
    const showApplyYaml = applyYaml && applyYaml.length >= 2;

    // Format toDelete portion of YAML.
    let toDeleteSection;
    if (shouldDelete && toDelete) {
        toDeleteSection = toDelete
            .map((entry) => `# kubectl -n ${entry.namespace} delete networkpolicy ${entry.name}`)
            .join('\n');
    }

    // Format complete YAML for display.
    let displayYaml;
    if (shouldDelete && showApplyYaml) {
        displayYaml = [toDeleteSection, applyYaml].join('\n---\n');
    } else if (shouldDelete && !showApplyYaml) {
        displayYaml = toDeleteSection;
    } else if (!shouldDelete && showApplyYaml) {
        displayYaml = applyYaml;
    } else {
        displayYaml = 'No policies need to be created or deleted.';
    }

    return (
        <Tabs headers={tabs}>
            <Tab>
                <div className="flex flex-col bg-base-100 overflow-auto h-full">
                    <pre className="p-3 pt-4 whitespace-pre-wrap word-break">{displayYaml}</pre>
                </div>
            </Tab>
        </Tabs>
    );
}

export default SuccessViewTabs;
