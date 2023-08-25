import React from 'react';
import { useLocation } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';
import {
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    Title,
    ToggleGroup,
    ToggleGroupItem,
} from '@patternfly/react-core';
import isEqual from 'lodash/isEqual';

import { vulnManagementImagesPath } from 'routePaths';
import useURLSearch from 'hooks/useURLSearch';
import useWidgetConfig from 'hooks/useWidgetConfig';
import { SearchFilter } from 'types/search';
import { getQueryString } from 'utils/queryStringUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import LinkShim from 'Components/PatternFly/LinkShim';
import WidgetCard from 'Components/PatternFly/WidgetCard';

import ImagesAtMostRiskTable, { CveStatusOption, ImageData } from './ImagesAtMostRiskTable';
import isResourceScoped from '../utils';
import NoDataEmptyState from './NoDataEmptyState';
import WidgetOptionsMenu from './WidgetOptionsMenu';
import WidgetOptionsResetButton from './WidgetOptionsResetButton';

function getTitle(searchFilter: SearchFilter, imageStatusOption: ImageStatusOption) {
    return imageStatusOption === 'Active' || isResourceScoped(searchFilter)
        ? 'Active images at most risk'
        : 'Images at most risk';
}

function getViewAllLink(searchFilter: SearchFilter) {
    const queryString = getQueryString({
        s: searchFilter,
        sort: [{ id: 'Image Risk Priority', desc: 'false' }],
    });
    return `${vulnManagementImagesPath}${queryString}`;
}

export const imagesAtMostRiskQuery = gql`
    query getImagesAtMostRisk($query: String) {
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
            imageVulnerabilityCounter {
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

type Config = {
    cveStatus: CveStatusOption;
    imageStatus: ImageStatusOption;
};

const defaultConfig: Config = {
    cveStatus: 'Fixable',
    imageStatus: 'All',
};

function ImagesAtMostRisk() {
    const { searchFilter } = useURLSearch();
    const { pathname } = useLocation();

    const [config, updateConfig] = useWidgetConfig<Config>(
        'ImagesAtMostRisk',
        pathname,
        defaultConfig
    );
    const { cveStatus, imageStatus } = config;

    const variables = getQueryVariables(searchFilter, imageStatus);
    const { data, previousData, loading, error } = useQuery<ImageData>(imagesAtMostRiskQuery, {
        variables,
    });

    const imageData = data || previousData;
    const isScopeApplied = isResourceScoped(searchFilter);
    const isOptionsChanged = !isEqual(config, defaultConfig);

    return (
        <WidgetCard
            isLoading={loading || !imageData}
            error={error}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">{getTitle(searchFilter, imageStatus)}</Title>
                    </FlexItem>
                    <FlexItem>
                        {isOptionsChanged && (
                            <WidgetOptionsResetButton onClick={() => updateConfig(defaultConfig)} />
                        )}
                        <WidgetOptionsMenu
                            bodyContent={
                                <Form>
                                    <FormGroup
                                        fieldId={`${fieldIdPrefix}-fixable`}
                                        label="Image vulnerabilities"
                                    >
                                        <ToggleGroup aria-label="Show all CVEs or fixable CVEs only">
                                            <ToggleGroupItem
                                                className="pf-u-font-weight-normal"
                                                text="Fixable CVEs"
                                                buttonId={`${fieldIdPrefix}-fixable-only`}
                                                isSelected={cveStatus === 'Fixable'}
                                                onChange={() =>
                                                    updateConfig({ cveStatus: 'Fixable' })
                                                }
                                            />
                                            <ToggleGroupItem
                                                text="All CVEs"
                                                buttonId={`${fieldIdPrefix}-all-cves`}
                                                isSelected={cveStatus === 'All'}
                                                onChange={() => updateConfig({ cveStatus: 'All' })}
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
                                                    imageStatus === 'Active' || isScopeApplied
                                                }
                                                onChange={() =>
                                                    updateConfig({ imageStatus: 'Active' })
                                                }
                                            />
                                            <ToggleGroupItem
                                                text="All images"
                                                buttonId={`${fieldIdPrefix}-status-all`}
                                                isSelected={
                                                    imageStatus === 'All' && !isScopeApplied
                                                }
                                                isDisabled={isScopeApplied}
                                                onChange={() =>
                                                    updateConfig({ imageStatus: 'All' })
                                                }
                                            />
                                        </ToggleGroup>
                                    </FormGroup>
                                </Form>
                            }
                        />
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
                <ImagesAtMostRiskTable imageData={imageData} cveStatusOption={cveStatus} />
            ) : (
                <NoDataEmptyState />
            )}
        </WidgetCard>
    );
}

export default ImagesAtMostRisk;
