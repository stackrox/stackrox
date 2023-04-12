import React from 'react';
import {
    PageSection,
    Title,
    Divider,
    Toolbar,
    ToolbarItem,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import useLocalStorage from 'hooks/useLocalStorage';
import PageTitle from 'Components/PageTitle';
import CveStatusTabNavigation from './CveStatusTabNavigation';
import DefaultFilterModal from '../DefaultFilterModal';
import { VulnMgmtLocalStorage } from '../types';

const emptyStorage: VulnMgmtLocalStorage = {
    preferences: {
        defaultFilters: {
            Severity: [],
            Fixable: [],
        },
    },
};

function WorkloadCvesOverviewPage() {
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
                <CveStatusTabNavigation defaultFilters={storedValue.preferences.defaultFilters} />
            </PageSection>
        </>
    );
}

export default WorkloadCvesOverviewPage;
