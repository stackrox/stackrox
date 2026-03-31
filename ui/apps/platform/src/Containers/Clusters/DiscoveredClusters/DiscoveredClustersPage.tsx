import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Content,
    PageSection,
    Spinner,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import usePermissions from 'hooks/usePermissions';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { fetchCloudSources } from 'services/CloudSourceService';
import {
    countDiscoveredClusters,
    defaultSortOption,
    getListDiscoveredClustersArg,
    listDiscoveredClusters,
    sortFields,
} from 'services/DiscoveredClusterService';
import type { DiscoveredCluster } from 'services/DiscoveredClusterService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { clustersBasePath } from 'routePaths';

import DiscoveredClustersTable from './DiscoveredClustersTable';
import DiscoveredClustersToolbar from './DiscoveredClustersToolbar';

const title = 'Discovered clusters';

function DiscoveredClustersPage(): ReactElement {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { getSortParams, sortOption } = useURLSort({ defaultSortOption, sortFields });

    const [count, setCount] = useState(0);
    const [errorMessage, setErrorMessage] = useState('');
    const [clusters, setClusters] = useState<DiscoveredCluster[]>([]);
    // Use currentDatetime === null as substitute for initial isLoading.
    const [currentDatetime, setCurrentDatetime] = useState<Date | null>(null);
    const [isReloading, setIsReloading] = useState(false);

    const { hasReadAccess } = usePermissions();
    const [sourceNameMap, setSourceNameMap] = useState<Map<string, string>>(new Map());

    const hasReadAccessForIntegration = hasReadAccess('Integration');
    useEffect(() => {
        if (hasReadAccessForIntegration) {
            fetchCloudSources()
                .then(({ response: { cloudSources } }) => {
                    setSourceNameMap(new Map(cloudSources.map(({ id, name }) => [id, name])));
                })
                .catch(() => {
                    // TODO
                });
        }
    }, [hasReadAccessForIntegration]);

    useEffect(() => {
        setIsReloading(true);
        const listArg = getListDiscoveredClustersArg({ page, perPage, searchFilter, sortOption });
        const { filter } = listArg;

        Promise.all([countDiscoveredClusters(filter), listDiscoveredClusters(listArg)])
            .then(([countFromResponse, clustersFromResponse]) => {
                setClusters(clustersFromResponse);
                setCount(countFromResponse);
                setErrorMessage('');
            })
            .catch((error) => {
                setClusters([]);
                setCount(0);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setCurrentDatetime(new Date());
                setIsReloading(false);
            });
    }, [page, perPage, searchFilter, sortOption]);

    return (
        <>
            <PageTitle title={title} />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={clustersBasePath}>Clusters</BreadcrumbItemLink>
                    <BreadcrumbItem isActive>{title}</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <Title headingLevel="h1">{title}</Title>
                <Content component="p">
                    Discovered clusters might not yet have secured cluster services.
                </Content>
            </PageSection>
            <PageSection>
                {currentDatetime === null ? (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                ) : errorMessage ? (
                    <Alert
                        variant="warning"
                        title="Unable to fetch discovered clusters"
                        component="p"
                        isInline
                    >
                        {errorMessage}
                    </Alert>
                ) : (
                    <>
                        <DiscoveredClustersToolbar
                            count={count}
                            isDisabled={isReloading}
                            page={page}
                            perPage={perPage}
                            setPage={setPage}
                            setPerPage={setPerPage}
                            searchFilter={searchFilter}
                            setSearchFilter={setSearchFilter}
                        />
                        <DiscoveredClustersTable
                            clusters={clusters}
                            currentDatetime={currentDatetime}
                            getSortParams={getSortParams}
                            searchFilter={searchFilter}
                            sourceNameMap={sourceNameMap}
                        />
                    </>
                )}
            </PageSection>
        </>
    );
}

export default DiscoveredClustersPage;
