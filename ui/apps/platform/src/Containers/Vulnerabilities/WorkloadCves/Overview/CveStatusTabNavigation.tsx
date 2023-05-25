import React from 'react';
import { useQuery } from '@apollo/client';
import {
    Tabs,
    Tab,
    TabTitleText,
    TabsComponent,
    PageSection,
    Card,
    CardBody,
} from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLSearch from 'hooks/useURLSearch';
import ImagesTableContainer from './ImagesTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import CVEsTableContainer from './CVEsTableContainer';
import { entityTypeCountsQuery } from '../components/EntityTypeToggleGroup';
import { DefaultFilters, cveStatusTabValues, entityTabValues } from '../types';
import { getCveStatusScopedQueryString, parseQuerySearchFilter } from '../searchUtils';

type CveStatusTabNavigationProps = {
    defaultFilters: DefaultFilters;
};

function CveStatusTabNavigation({ defaultFilters }: CveStatusTabNavigationProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [activeCVEStatusKey, setActiveCVEStatusKey] = useURLStringUnion(
        'cveStatus',
        cveStatusTabValues
    );
    const [activeEntityTabKey] = useURLStringUnion('entityTab', entityTabValues);

    function handleTabClick(e, tab) {
        setActiveCVEStatusKey(tab);
    }

    const { data: countsData = { imageCount: 0, imageCVECount: 0, deploymentCount: 0 } } = useQuery(
        entityTypeCountsQuery,
        {
            variables: {
                query: getCveStatusScopedQueryString(querySearchFilter, activeCVEStatusKey),
            },
        }
    );

    return (
        <Tabs
            activeKey={activeCVEStatusKey}
            onSelect={handleTabClick}
            component={TabsComponent.nav}
            className="pf-u-pl-lg pf-u-background-color-100"
            mountOnEnter
            unmountOnExit
        >
            <Tab eventKey="Observed" title={<TabTitleText>Observed CVEs</TabTitleText>}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTableContainer
                                    defaultFilters={defaultFilters}
                                    countsData={countsData}
                                    cveStatusTab={activeCVEStatusKey}
                                />
                            )}
                            {activeEntityTabKey === 'Image' && (
                                <ImagesTableContainer
                                    defaultFilters={defaultFilters}
                                    countsData={countsData}
                                    cveStatusTab={activeCVEStatusKey}
                                />
                            )}
                            {activeEntityTabKey === 'Deployment' && (
                                <DeploymentsTableContainer
                                    defaultFilters={defaultFilters}
                                    countsData={countsData}
                                    cveStatusTab={activeCVEStatusKey}
                                />
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </Tab>
            <Tab eventKey="Deferred" title={<TabTitleText>Deferrals</TabTitleText>} isDisabled>
                deferrals tbd
            </Tab>
            <Tab
                eventKey="False Positive"
                title={<TabTitleText>False Positives</TabTitleText>}
                isDisabled
            >
                False-positives tbd
            </Tab>
        </Tabs>
    );
}

export default CveStatusTabNavigation;
