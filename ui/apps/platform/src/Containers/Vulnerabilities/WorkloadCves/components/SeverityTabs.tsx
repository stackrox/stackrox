import { Badge, Tab, TabTitleIcon, TabTitleText, Tabs } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import useURLStringUnion from 'hooks/useURLStringUnion';
import { severityTabValues } from '../../types';
import type { SeverityTab } from '../../types';
import { severityLabelToSeverity } from '../../utils/searchUtils';

export const severityTabContentId = 'severity-tab-content';

export type SeverityTabsProps = {
    counts?: Partial<Record<SeverityTab, number>>;
    onChange?: (severity: SeverityTab) => void;
};

function SeverityTabs({ counts, onChange }: SeverityTabsProps) {
    const [activeSeverityTab, setActiveSeverityTab] = useURLStringUnion(
        'severityTab',
        severityTabValues
    );

    return (
        <Tabs
            activeKey={activeSeverityTab}
            onSelect={(_e, tab) => {
                setActiveSeverityTab(tab);
                if (
                    onChange &&
                    severityTabValues.includes(tab as SeverityTab) &&
                    tab !== activeSeverityTab
                ) {
                    onChange(tab as SeverityTab);
                }
            }}
            isBox
        >
            {severityTabValues.map((severity) => {
                const sevEnum = severityLabelToSeverity(severity);
                const Icon = SeverityIcons[sevEnum];
                const count = counts?.[severity];

                return (
                    <Tab
                        key={severity}
                        eventKey={severity}
                        title={
                            <>
                                <TabTitleIcon>
                                    <Icon />
                                </TabTitleIcon>
                                <TabTitleText>{severity}</TabTitleText>
                                {count !== undefined && (
                                    <Badge isRead>{count.toLocaleString()}</Badge>
                                )}
                            </>
                        }
                        tabContentId={severityTabContentId}
                    />
                );
            })}
        </Tabs>
    );
}

export default SeverityTabs;
