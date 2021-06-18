/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useState } from 'react';
import { Tab, TabContent, Tabs, TabTitleText } from '@patternfly/react-core';

import { LabelSelector, LabelSelectorsKey } from 'services/RolesService';

import LabelSelectors from './LabelSelectors';

export type LabelInclusionProps = {
    clusterLabelSelectors: LabelSelector[];
    namespaceLabelSelectors: LabelSelector[];
    hasAction: boolean;
    handleChangeLabelSelectors: (
        labelSelectorsKey: LabelSelectorsKey,
        labelSelectorsNext: LabelSelector[]
    ) => void;
};

function LabelInclusion({
    clusterLabelSelectors,
    namespaceLabelSelectors,
    hasAction,
    handleChangeLabelSelectors,
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
                    title={<TabTitleText>Cluster</TabTitleText>}
                />
                <Tab
                    eventKey="namespaceLabelSelectors"
                    tabContentId="namespaceLabelSelectors"
                    title={<TabTitleText>Namespace</TabTitleText>}
                />
            </Tabs>
            <TabContent
                eventKey="clusterLabelSelectors"
                id="clusterLabelSelectors"
                hidden={activeKeyTab !== 'clusterLabelSelectors'}
            >
                <LabelSelectors
                    labelSelectors={clusterLabelSelectors}
                    labelSelectorsKey="clusterLabelSelectors"
                    hasAction={hasAction}
                    handleChangeLabelSelectors={handleChangeLabelSelectors}
                />
            </TabContent>
            <TabContent
                eventKey="namespaceLabelSelectors"
                id="namespaceLabelSelectors"
                hidden={activeKeyTab !== 'namespaceLabelSelectors'}
            >
                <LabelSelectors
                    labelSelectors={namespaceLabelSelectors}
                    labelSelectorsKey="namespaceLabelSelectors"
                    hasAction={hasAction}
                    handleChangeLabelSelectors={handleChangeLabelSelectors}
                />
            </TabContent>
        </>
    );
}

export default LabelInclusion;
