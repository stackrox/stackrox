import React, { ReactElement, useState } from 'react';
import { Badge, Tab, TabContent, Tabs, TabTitleText } from '@patternfly/react-core';

import { LabelSelector, LabelSelectorsKey } from 'services/AccessScopesService';

import { LabelSelectorsEditingState } from './accessScopes.utils';
import LabelSelectorCards from './LabelSelectorCards';

export type LabelInclusionProps = {
    clusterLabelSelectors: LabelSelector[];
    namespaceLabelSelectors: LabelSelector[];
    hasAction: boolean;
    labelSelectorsEditingState: LabelSelectorsEditingState;
    setLabelSelectorsEditingState: (nextState: LabelSelectorsEditingState) => void;
    handleLabelSelectorsChange: (
        labelSelectorsKey: LabelSelectorsKey,
        labelSelectorsNext: LabelSelector[]
    ) => void;
};

function LabelInclusion({
    clusterLabelSelectors,
    namespaceLabelSelectors,
    hasAction,
    labelSelectorsEditingState,
    setLabelSelectorsEditingState,
    handleLabelSelectorsChange,
}: LabelInclusionProps): ReactElement {
    const [activeKeyTab, setActiveKeyTab] = useState('clusterLabelSelectors');

    function onSelectTab(event, eventKey) {
        event.preventDefault(); // without this, the page refreshes with empty query string :(
        setActiveKeyTab(eventKey);
    }

    return (
        <>
            <Tabs activeKey={activeKeyTab} onSelect={onSelectTab}>
                <Tab
                    eventKey="clusterLabelSelectors"
                    tabContentId="clusterLabelSelectors"
                    title={
                        <TabTitleText>
                            Cluster
                            <Badge isRead className="pf-v5-u-ml-sm">
                                {clusterLabelSelectors.length}
                            </Badge>
                        </TabTitleText>
                    }
                />
                <Tab
                    eventKey="namespaceLabelSelectors"
                    tabContentId="namespaceLabelSelectors"
                    title={
                        <TabTitleText>
                            Namespace
                            <Badge isRead className="pf-v5-u-ml-sm">
                                {namespaceLabelSelectors.length}
                            </Badge>
                        </TabTitleText>
                    }
                />
            </Tabs>
            <TabContent
                eventKey="clusterLabelSelectors"
                id="clusterLabelSelectors"
                hidden={activeKeyTab !== 'clusterLabelSelectors'}
            >
                <LabelSelectorCards
                    labelSelectors={clusterLabelSelectors}
                    labelSelectorsKey="clusterLabelSelectors"
                    hasAction={hasAction}
                    labelSelectorsEditingState={labelSelectorsEditingState}
                    setLabelSelectorsEditingState={setLabelSelectorsEditingState}
                    handleLabelSelectorsChange={handleLabelSelectorsChange}
                />
            </TabContent>
            <TabContent
                eventKey="namespaceLabelSelectors"
                id="namespaceLabelSelectors"
                hidden={activeKeyTab !== 'namespaceLabelSelectors'}
            >
                <LabelSelectorCards
                    labelSelectors={namespaceLabelSelectors}
                    labelSelectorsKey="namespaceLabelSelectors"
                    hasAction={hasAction}
                    labelSelectorsEditingState={labelSelectorsEditingState}
                    setLabelSelectorsEditingState={setLabelSelectorsEditingState}
                    handleLabelSelectorsChange={handleLabelSelectorsChange}
                />
            </TabContent>
        </>
    );
}

export default LabelInclusion;
