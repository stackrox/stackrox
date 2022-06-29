import React, { useState, useEffect } from 'react';
import {
    PageSection,
    Bullseye,
    Spinner,
    Alert,
    Divider,
    Button,
    Flex,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getPolicyCategories } from 'services/PoliciesService';
import PolicyManagementHeader from 'Containers/PolicyManagement/PolicyManagementHeader';
import PolicyCategoriesListSection from './PolicyCategoriesListSection';

function PolicyCategoriesPage(): React.ReactElement {
    const [isLoading, setIsLoading] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');
    // TODO switch type once new API is in
    const [policyCategories, setPolicyCategories] = useState<string[]>([]);

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

    if (!isLoading && !errorMessage && policyCategories.length > 0) {
        pageContent = <PolicyCategoriesListSection policyCategories={policyCategories} />;
    }

    useEffect(() => {
        setIsLoading(true);
        getPolicyCategories()
            .then((categories) => {
                setPolicyCategories(categories);
                setErrorMessage('');
            })
            .catch((error) => {
                setPolicyCategories([]);
                setErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => setIsLoading(false));
    }, []);

    return (
        <>
            <PolicyManagementHeader currentTabTitle="Policy categories" />
            <Divider component="div" />
            <PageSection variant="light" className="pf-u-py-0">
                <Toolbar inset={{ default: 'insetNone' }}>
                    <ToolbarContent>
                        <ToolbarItem>
                            <div className="pf-u-font-size-sm">
                                Manage categories for your policies.
                            </div>
                        </ToolbarItem>
                        <ToolbarItem alignment={{ default: 'alignRight' }}>
                            <Flex>
                                <Button variant="primary" onClick={() => {}}>
                                    Create category
                                </Button>
                            </Flex>
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <Divider component="div" />
            {pageContent}
        </>
    );
}

export default PolicyCategoriesPage;
