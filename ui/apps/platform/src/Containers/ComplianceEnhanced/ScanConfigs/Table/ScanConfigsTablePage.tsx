/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import { Alert, Divider, Bullseye, Button, PageSection, Spinner } from '@patternfly/react-core';

import { complianceEnhancedScanConfigsPath } from 'routePaths';
import { getScanConfigs, ScanConfig } from 'services/ComplianceEnhancedService';
import { SearchFilter } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import ScanConfigsHeader from '../ScanConfigsHeader';

type ScanConfigsTablePageProps = {
    hasWriteAccessForCompliance: boolean;
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    searchFilter?: SearchFilter;
};

function ScanConfigsTablePage({
    hasWriteAccessForCompliance,
    handleChangeSearchFilter,
    searchFilter,
}: ScanConfigsTablePageProps): React.ReactElement {
    const history = useHistory();

    const [isLoading, setIsLoading] = useState(false);
    const [scanSchedules, setScanSchedules] = useState<ScanConfig[]>([]);
    const [errorMessage, setErrorMessage] = useState('');

    const [searchOptions, setSearchOptions] = useState<string[]>([]);

    function onClickCreate() {
        history.push(`${complianceEnhancedScanConfigsPath}/?action=create`);
    }

    function fetchScanSchedules(query: string) {
        setIsLoading(true);
        getScanConfigs(query)
            .then((data) => {
                setScanSchedules(data);
                setErrorMessage('');
            })
            .catch((error) => {
                setScanSchedules([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoading(false));
    }

    const query = searchFilter ? getRequestQueryStringForSearchFilter(searchFilter) : '';

    useEffect(() => {
        fetchScanSchedules(query);
    }, [query]);

    let pageContent = (
        <PageSection variant="light" isFilled id="policies-table-loading">
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        </PageSection>
    );

    if (errorMessage) {
        pageContent = (
            <PageSection variant="light" isFilled id="policies-table-error">
                <Bullseye>
                    <Alert variant="danger" title={errorMessage} />
                </Bullseye>
            </PageSection>
        );
    }

    if (!isLoading && !errorMessage) {
        pageContent = <div>ScanSchedulesTable goes here</div>;
    }

    return (
        <>
            <ScanConfigsHeader
                actions={
                    hasWriteAccessForCompliance ? (
                        <>
                            <Button variant="primary" onClick={onClickCreate}>
                                Create scan schedule
                            </Button>
                        </>
                    ) : (
                        <></>
                    )
                }
                description="Configure scan schedules bound to clusters and policies."
            />
            <Divider component="div" />
            {pageContent}
        </>
    );
}

export default ScanConfigsTablePage;
