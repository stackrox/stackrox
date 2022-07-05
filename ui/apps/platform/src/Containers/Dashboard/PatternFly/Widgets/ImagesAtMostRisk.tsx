import React, { useState } from 'react';
import { gql, useQuery } from '@apollo/client';
import {
    Button,
    Dropdown,
    DropdownToggle,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';

import { vulnManagementImagesPath } from 'routePaths';
import useURLSearch from 'hooks/useURLSearch';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import LinkShim from 'Components/PatternFly/LinkShim';

import WidgetCard from './WidgetCard';
import ImagesAtMostRiskTable, { CveStatusOption, ImageData } from './ImagesAtMostRiskTable';
import isResourceScoped from '../utils';
import NoDataEmptyState from './NoDataEmptyState';

function getTitle(searchFilter: SearchFilter, imageStatusOption: ImageStatusOption) {
    return imageStatusOption === 'Active' || isResourceScoped(searchFilter)
        ? 'Active images at most risk'
        : 'All images at most risk';
}

function getViewAllLink(searchFilter: SearchFilter) {
    const queryString = getQueryString({
        s: searchFilter,
        sort: [{ id: 'Image Risk Priority', desc: 'false' }],
    });
    return `${vulnManagementImagesPath}${queryString}`;
}

export const imagesQuery = gql`
    query getImages($query: String) {
        images(
            query: $query
            pagination: { limit: 6, sortOption: { field: "Image Risk Priority", reversed: false } }
        ) {
            id
            name {
                remote
                fullName
            }
            priority
            vulnCounter {
                important {
                    total
                    fixable
                }
                critical {
                    total
                    fixable
                }
            }
        }
    }
`;

// If no resource scope is applied and the user selects "Active images" only, we
// can use the wildcard query `Namespace:*` to return images part of any namespace i.e. active
function getQueryVariables(searchFilter: SearchFilter, statusOption: ImageStatusOption) {
    const query =
        statusOption === 'Active' && !isResourceScoped(searchFilter)
            ? 'Namespace:*'
            : getRequestQueryStringForSearchFilter(searchFilter);
    return { query };
}

const fieldIdPrefix = 'images-at-most-risk';

type ImageStatusOption = 'Active' | 'All';

function ImagesAtMostRisk() {
    const { isOpen: isOptionsOpen, onToggle: toggleOptionsOpen } = useSelectToggle();
    const { searchFilter } = useURLSearch();

    const [cveStatusOption, setCveStatusOption] = useState<CveStatusOption>('Fixable');
    const [imageStatusOption, setImageStatusOption] = useState<ImageStatusOption>('All');

    const variables = getQueryVariables(searchFilter, imageStatusOption);
    const { data, previousData, loading, error } = useQuery<ImageData>(imagesQuery, {
        variables,
    });

    const imageData = data || previousData;
    const isScopeApplied = isResourceScoped(searchFilter);

    return (
        <WidgetCard
            isLoading={loading || !imageData}
            error={error}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">{getTitle(searchFilter, imageStatusOption)}</Title>
                    </FlexItem>
                    <FlexItem>
                        <Dropdown
                            className="pf-u-mr-sm"
                            toggle={
                                <DropdownToggle
                                    id={`${fieldIdPrefix}-options-toggle`}
                                    toggleVariant="secondary"
                                    onToggle={toggleOptionsOpen}
                                >
                                    Options
                                </DropdownToggle>
                            }
                            position="right"
                            isOpen={isOptionsOpen}
                        >
                            <Form className="pf-u-px-md pf-u-py-sm" style={{ minWidth: '250px' }}>
                                <FormGroup
                                    fieldId={`${fieldIdPrefix}-fixable`}
                                    label="Image vulnerabilities"
                                >
                                    <ToggleGroup aria-label="Show all CVEs or fixable CVEs only">
                                        <ToggleGroupItem
                                            className="pf-u-font-weight-normal"
                                            text="Fixable CVEs"
                                            buttonId={`${fieldIdPrefix}-fixable-only`}
                                            isSelected={cveStatusOption === 'Fixable'}
                                            onChange={() => setCveStatusOption('Fixable')}
                                        />
                                        <ToggleGroupItem
                                            text="All CVEs"
                                            buttonId={`${fieldIdPrefix}-all-cves`}
                                            isSelected={cveStatusOption === 'All'}
                                            onChange={() => setCveStatusOption('All')}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                                <FormGroup
                                    fieldId={`${fieldIdPrefix}-lifecycle`}
                                    label="Image status"
                                >
                                    <ToggleGroup aria-label="Show all images or active images only">
                                        <ToggleGroupItem
                                            text="Active images"
                                            buttonId={`${fieldIdPrefix}-status-active`}
                                            isSelected={
                                                imageStatusOption === 'Active' || isScopeApplied
                                            }
                                            onChange={() => setImageStatusOption('Active')}
                                        />
                                        <ToggleGroupItem
                                            text="All images"
                                            buttonId={`${fieldIdPrefix}-status-all`}
                                            isSelected={
                                                imageStatusOption === 'All' && !isScopeApplied
                                            }
                                            isDisabled={isScopeApplied}
                                            onChange={() => setImageStatusOption('All')}
                                        />
                                    </ToggleGroup>
                                </FormGroup>
                            </Form>
                        </Dropdown>
                        <Button
                            variant="secondary"
                            component={LinkShim}
                            href={getViewAllLink(searchFilter)}
                        >
                            View all
                        </Button>
                    </FlexItem>
                </Flex>
            }
        >
            {imageData && imageData.images.length > 0 ? (
                <ImagesAtMostRiskTable imageData={imageData} cveStatusOption={cveStatusOption} />
            ) : (
                <NoDataEmptyState />
            )}
        </WidgetCard>
    );
}

export default ImagesAtMostRisk;
