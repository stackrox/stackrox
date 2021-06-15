/* eslint-disable react/jsx-no-bind */
import React, { ReactElement, useState } from 'react';
import { Tab, TabContent, Tabs, TabTitleText } from '@patternfly/react-core';

import { LabelSelector } from 'services/RolesService';

export type LabelInclusionProps = {
    clusterLabelSelectors: LabelSelector[];
    namespaceLabelSelectors: LabelSelector[];
    hasAction: boolean;
};

function LabelInclusion({
    clusterLabelSelectors,
    namespaceLabelSelectors,
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
                    title={<TabTitleText>Cluster label selectors</TabTitleText>}
                />
                <Tab
                    eventKey="namespaceLabelSelectors"
                    tabContentId="namespaceLabelSelectors"
                    title={<TabTitleText>Namespace label selectors</TabTitleText>}
                />
            </Tabs>
            <TabContent
                eventKey="clusterLabelSelectors"
                id="clusterLabelSelectors"
                hidden={activeKeyTab !== 'clusterLabelSelectors'}
            >
                {JSON.stringify(clusterLabelSelectors, null, 2)}
            </TabContent>
            <TabContent
                eventKey="namespaceLabelSelectors"
                id="namespaceLabelSelectors"
                hidden={activeKeyTab !== 'namespaceLabelSelectors'}
            >
                {JSON.stringify(namespaceLabelSelectors, null, 2)}
            </TabContent>
        </>
    );
}

export default LabelInclusion;
