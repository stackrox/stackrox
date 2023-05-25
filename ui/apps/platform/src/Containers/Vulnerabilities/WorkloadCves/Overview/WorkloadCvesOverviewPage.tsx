import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Toolbar,
    ToolbarItem,
    Flex,
    FlexItem,
    Card,
    CardBody,
} from '@patternfly/react-core';
import { useQuery } from '@apollo/client';

import useLocalStorage from 'hooks/useLocalStorage';
import useURLSearch from 'hooks/useURLSearch';
import useURLStringUnion from 'hooks/useURLStringUnion';
import PageTitle from 'Components/PageTitle';

import { VulnMgmtLocalStorage, entityTabValues } from '../types';
import { parseQuerySearchFilter, getCveStatusScopedQueryString } from '../searchUtils';
import DefaultFilterModal from '../components/DefaultFilterModal';
import { entityTypeCountsQuery } from '../components/EntityTypeToggleGroup';
import CVEsTableContainer from './CVEsTableContainer';
import DeploymentsTableContainer from './DeploymentsTableContainer';
import ImagesTableContainer from './ImagesTableContainer';

const emptyStorage: VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: {
            Severity: [],
            Fixable: [],
        },
    },
};

function WorkloadCvesOverviewPage() {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const [activeEntityTabKey] = useURLStringUnion('entityTab', entityTabValues);

    const { data: countsData = { imageCount: 0, imageCVECount: 0, deploymentCount: 0 } } = useQuery(
        entityTypeCountsQuery,
        {
            variables: {
                query: getCveStatusScopedQueryString(querySearchFilter),
            },
        }
    );

    const [storedValue, setStoredValue] = useLocalStorage('vulnerabilityManagement', emptyStorage);

    function setLocalStorage(values) {
        setStoredValue({
            preferences: {
                defaultFilters: values,
            },
        });
    }

    return (
        <>
            <PageTitle title="Workload CVEs Overview" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Toolbar>
                    <ToolbarItem alignment={{ default: 'alignRight' }}>
                        <DefaultFilterModal
                            defaultFilters={storedValue.preferences.defaultFilters}
                            setLocalStorage={setLocalStorage}
                        />
                    </ToolbarItem>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light" padding={{ default: 'noPadding' }}>
                <Flex direction={{ default: 'column' }} className="pf-u-py-lg pf-u-pl-lg">
                    <FlexItem>
                        <Title headingLevel="h1">Workload CVEs</Title>
                    </FlexItem>
                    <FlexItem>
                        Prioritize and manage scanned CVEs across images and deployments
                    </FlexItem>
                </Flex>
            </PageSection>
            <PageSection padding={{ default: 'noPadding' }}>
                <PageSection isCenterAligned>
                    <Card>
                        <CardBody>
                            {activeEntityTabKey === 'CVE' && (
                                <CVEsTableContainer
                                    defaultFilters={storedValue.preferences.defaultFilters}
                                    countsData={countsData}
                                />
                            )}
                            {activeEntityTabKey === 'Image' && (
                                <ImagesTableContainer
                                    defaultFilters={storedValue.preferences.defaultFilters}
                                    countsData={countsData}
                                />
                            )}
                            {activeEntityTabKey === 'Deployment' && (
                                <DeploymentsTableContainer
                                    defaultFilters={storedValue.preferences.defaultFilters}
                                    countsData={countsData}
                                />
                            )}
                        </CardBody>
                    </Card>
                </PageSection>
            </PageSection>
        </>
    );
}

export default WorkloadCvesOverviewPage;
