import React from 'react';
import {
    Tabs,
    TabsComponent,
    Tab,
    TabTitleText,
    TabProps,
    TabsProps,
} from '@patternfly/react-core';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { cveStatusTabValues } from '../types';

function makeBaseTab(eventKey: (typeof cveStatusTabValues)[number], title: string) {
    const TabComponent = (props: Omit<TabProps, 'ref' | 'title' | 'eventKey'>) => {
        return (
            <Tab
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                {...props}
                title={<TabTitleText>{title}</TabTitleText>}
                eventKey={eventKey}
            >
                {props.children}
            </Tab>
        );
    };

    TabComponent.defaultProps = { eventKey, title };

    return TabComponent;
}

export const ObservedCvesTab = makeBaseTab('Observed', 'Observed CVEs');
export const DeferredCvesTab = makeBaseTab('Deferred', 'Deferrals');
export const FalsePositiveCvesTab = makeBaseTab('False Positive', 'False positives');

export type CveStatusTabsProps = Omit<TabsProps, 'children' | 'ref'> & {
    children: [
        React.ReactElement<typeof ObservedCvesTab>,
        React.ReactElement<typeof DeferredCvesTab>,
        React.ReactElement<typeof FalsePositiveCvesTab>
    ];
};

function CveStatusTabs(props: CveStatusTabsProps) {
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('cveStatus', cveStatusTabValues);
    return (
        <Tabs
            activeKey={activeTabKey}
            onSelect={(e, key) => setActiveTabKey(key)}
            component={TabsComponent.nav}
            mountOnEnter
            unmountOnExit
            {...props}
        >
            {props.children}
        </Tabs>
    );
}

export default CveStatusTabs;
