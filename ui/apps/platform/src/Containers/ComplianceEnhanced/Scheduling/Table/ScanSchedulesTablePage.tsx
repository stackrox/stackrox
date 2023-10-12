/* eslint-disable @typescript-eslint/no-unused-vars */
import React, { useState, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import { PageSection, Bullseye, Alert, Spinner, Divider } from '@patternfly/react-core';

import { complianceEnhancedBasePath } from 'routePaths';
import { getScanSchedules, ScanSchedule } from 'services/ComplianceEnhancedService';
import { SearchFilter } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

type ScanSchedulesTablePageProps = {
    hasWriteAccessForCompliance: boolean;
    handleChangeSearchFilter: (searchFilter: SearchFilter) => void;
    searchFilter?: SearchFilter;
};

function ScanSchedulesTablePage({
    hasWriteAccessForCompliance,
    handleChangeSearchFilter,
    searchFilter,
}: ScanSchedulesTablePageProps): React.ReactElement {
    const history = useHistory();

    const [isLoading, setIsLoading] = useState(false);
    const [scanSchedules, setScanSchedules] = useState<ScanSchedule[]>([]);
    const [errorMessage, setErrorMessage] = useState('');

    const [searchOptions, setSearchOptions] = useState<string[]>([]);

    function onClickCreateScanScedule() {
        history.push(`${complianceEnhancedBasePath}/?action=create`);
    }

    function fetchScanSchedules(query: string) {
        setIsLoading(true);
        getScanSchedules(query)
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
            <div>header goes here</div>
            <Divider component="div" />
            {pageContent}
        </>
    );
}

export default ScanSchedulesTablePage;
