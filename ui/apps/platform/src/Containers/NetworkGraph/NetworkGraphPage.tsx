import React, { useState, useEffect } from 'react';
import { PageSection, Title, Flex, FlexItem } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import useURLSearch from 'hooks/useURLSearch';
import { getSearchOptionsForCategory } from 'services/SearchService';
import NetworkGraphToolbar from './NetworkGraphToolbar';
import NetworkGraph from './NetworkGraph';

import './NetworkGraphPage.css';

function NetworkGraphPage() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [searchOptions, setSearchOptions] = useState<string[]>([]);

    useEffect(() => {
        const { request, cancel } = getSearchOptionsForCategory('DEPLOYMENTS');
        request
            .then((options) => {
                setSearchOptions(options);
            })
            .catch(() => {
                // TODO
            });

        // eslint-disable-next-line @typescript-eslint/no-unsafe-return
        return cancel;
    }, []);

    return (
        <>
            <PageTitle title="Network Graph" />
            <PageSection variant="light">
                <Flex alignItems={{ default: 'alignItemsCenter' }}>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Network Graph</Title>
                    </FlexItem>
                </Flex>
                <NetworkGraphToolbar
                    searchFilter={searchFilter}
                    handleChangeSearchFilter={setSearchFilter}
                    searchOptions={searchOptions}
                />
            </PageSection>
            <PageSection className="network-graph no-padding">
                <NetworkGraph searchFilter={searchFilter} />
            </PageSection>
        </>
    );
}

export default NetworkGraphPage;
