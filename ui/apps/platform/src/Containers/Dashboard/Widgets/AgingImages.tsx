import React from 'react';
import { useLocation } from 'react-router-dom';
import {
    Flex,
    FlexItem,
    Title,
    Button,
    Form,
    FormGroup,
    Checkbox,
    TextInput,
    ValidatedOptions,
} from '@patternfly/react-core';
import { useQuery, gql } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';
import isEqual from 'lodash/isEqual';
import pluralize from 'pluralize';

import LinkShim from 'Components/PatternFly/LinkShim';
import useURLSearch from 'hooks/useURLSearch';
import useWidgetConfig from 'hooks/useWidgetConfig';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { SearchFilter } from 'types/search';
import { vulnManagementImagesPath } from 'routePaths';
import { getQueryString } from 'utils/queryStringUtils';
import WidgetCard from 'Components/PatternFly/WidgetCard';
import AgingImagesChart, {
    TimeRangeCounts,
    TimeRangeTupleIndex,
    TimeRangeTuple,
    timeRangeTupleIndices,
    getTimeFilterOption,
} from './AgingImagesChart';
import isResourceScoped from '../utils';
import NoDataEmptyState from './NoDataEmptyState';
import WidgetOptionsMenu from './WidgetOptionsMenu';
import WidgetOptionsResetButton from './WidgetOptionsResetButton';

export const imageCountQuery = gql`
    query agingImagesQuery($query0: String, $query1: String, $query2: String, $query3: String) {
        timeRange0: imageCount(query: $query0)
        timeRange1: imageCount(query: $query1)
        timeRange2: imageCount(query: $query2)
        timeRange3: imageCount(query: $query3)
    }
`;

function queryStringFor(searchFilter: SearchFilter, ageRange: number, nextAgeRange?: number) {
    const timeFilter = getTimeFilterOption(ageRange, nextAgeRange);

    return getRequestQueryStringForSearchFilter({
        ...searchFilter,
        'Image Created Time': timeFilter,
    });
}

type QueryVariables = Record<`query${TimeRangeTupleIndex}`, string>;

function getQueryVariables(timeRanges: TimeRangeTuple, searchFilter: SearchFilter): QueryVariables {
    return {
        query0: queryStringFor(searchFilter, timeRanges[0].value, timeRanges[1].value),
        query1: queryStringFor(searchFilter, timeRanges[1].value, timeRanges[2].value),
        query2: queryStringFor(searchFilter, timeRanges[2].value, timeRanges[3].value),
        query3: queryStringFor(searchFilter, timeRanges[3].value),
    };
}

// Gets the header string title for the widget based on applied filters and resulting counts
function getWidgetTitle(
    searchFilter: SearchFilter,
    selectedTimeRanges: TimeRangeTuple,
    timeRangeCounts?: TimeRangeCounts
): string {
    if (!timeRangeCounts) {
        return 'Aging images';
    }

    // The total number of images is obtained by finding the first enabled bucket
    // and getting the sum of that bucket and all following buckets.
    let totalImages = 0;
    const firstEnabledIndex = selectedTimeRanges.findIndex(({ enabled }) => enabled);
    const rangesToCount = Object.values(timeRangeCounts).slice(
        firstEnabledIndex,
        selectedTimeRanges.length
    );
    rangesToCount.forEach((range) => {
        totalImages += range;
    });

    const isActiveImages = isResourceScoped(searchFilter);

    if (isActiveImages) {
        return `${totalImages} Active aging ${pluralize('image', totalImages)}`;
    }
    return `${totalImages} Aging ${pluralize('image', totalImages)}`;
}

type Config = {
    timeRanges: TimeRangeTuple;
};

const defaultConfig: Config = {
    timeRanges: [
        { enabled: true, value: 30 },
        { enabled: true, value: 90 },
        { enabled: true, value: 180 },
        { enabled: true, value: 365 },
    ],
};

type TimeRangeAction =
    | {
          type: 'toggle';
          index: TimeRangeTupleIndex;
      }
    | {
          type: 'update';
          index: TimeRangeTupleIndex;
          value: number;
      }
    | {
          type: 'reset';
      };

function timeRangeReducer(state: Config, action: TimeRangeAction): Config {
    const nextState = cloneDeep(state);
    const { timeRanges } = nextState;

    switch (action.type) {
        case 'toggle':
            timeRanges[action.index].enabled = !timeRanges[action.index].enabled;
            return nextState;
        case 'update':
            timeRanges[action.index].value = action.value;
            return nextState;
        case 'reset':
            return defaultConfig;
        default:
            return nextState;
    }
}

const maxTimeRange = 366;

// Tests if a user entered value in the options menu is a valid number and falls within
// the range of the previous and following time range values in the list.
function isNumberInRange(timeRanges: TimeRangeTuple, index: TimeRangeTupleIndex): boolean {
    const { value } = timeRanges[index];
    const rangeValues = timeRanges.map((r) => r.value);
    const lowerBounds = [0, ...rangeValues.slice(0, 3)];
    const upperBounds = [...rangeValues.slice(1, 4), maxTimeRange];

    return value > lowerBounds[index] && value < upperBounds[index];
}

const fieldIdPrefix = 'aging-images';

function getViewAllLink(searchFilter: SearchFilter) {
    const queryString = getQueryString({
        s: searchFilter,
        sort: [{ id: 'Image Created Time', desc: 'false' }],
    });
    return `${vulnManagementImagesPath}${queryString}`;
}

function AgingImages() {
    const { searchFilter } = useURLSearch();
    const { pathname } = useLocation();

    const [{ timeRanges }, dispatch] = useWidgetConfig<Config, TimeRangeAction>(
        'AgingImages',
        pathname,
        defaultConfig,
        timeRangeReducer
    );

    const variables = getQueryVariables(timeRanges, searchFilter);
    const { data, previousData, loading, error } = useQuery<TimeRangeCounts>(imageCountQuery, {
        variables,
    });
    const timeRangeCounts = data ?? previousData;
    const isCountsDataEmpty =
        timeRangeCounts &&
        Object.values(timeRangeCounts).every(
            (value, index) => !timeRanges[index].enabled || value === 0
        );

    const isOptionsChanged = !isEqual(timeRanges, defaultConfig.timeRanges);

    let inputError: Error | undefined;
    let errorTitle: string | undefined;
    let errorMessage: string | undefined;
    if (timeRanges.every(({ enabled }) => !enabled)) {
        inputError = new Error('All image age ranges disabled');
        errorTitle = 'All image age ranges disabled';
        errorMessage = 'At least one image age range must be enabled in the options.';
    } else if (timeRangeTupleIndices.some((index) => !isNumberInRange(timeRanges, index))) {
        inputError = new Error('Invalid image ages');
        errorTitle = 'Incorrect image age values';
        errorMessage = 'There was an error retrieving data. Image ages must be in ascending order.';
    }

    return (
        <WidgetCard
            isLoading={loading && !timeRangeCounts}
            error={error || inputError}
            errorTitle={errorTitle}
            errorMessage={errorMessage}
            header={
                <Flex direction={{ default: 'row' }}>
                    <FlexItem grow={{ default: 'grow' }}>
                        <Title headingLevel="h2">
                            {getWidgetTitle(searchFilter, timeRanges, timeRangeCounts)}
                        </Title>
                    </FlexItem>
                    <FlexItem>
                        {isOptionsChanged && (
                            <WidgetOptionsResetButton onClick={() => dispatch({ type: 'reset' })} />
                        )}
                        <WidgetOptionsMenu
                            bodyContent={
                                <Form>
                                    <FormGroup
                                        fieldId={`${fieldIdPrefix}-time-range-0`}
                                        label="Image age values"
                                    >
                                        {timeRangeTupleIndices.map((index) => (
                                            <div key={index}>
                                                <Checkbox
                                                    aria-label="Toggle image time range"
                                                    id={`${fieldIdPrefix}-time-range-${index}`}
                                                    name={`${fieldIdPrefix}-time-range-${index}`}
                                                    className="pf-u-mb-sm pf-u-display-flex pf-u-align-items-center"
                                                    isChecked={timeRanges[index].enabled}
                                                    onChange={() =>
                                                        dispatch({ type: 'toggle', index })
                                                    }
                                                    label={
                                                        <TextInput
                                                            aria-label="Image age in days"
                                                            onChange={async (val) => {
                                                                const value = parseInt(val, 10);
                                                                if (!(value >= maxTimeRange)) {
                                                                    await dispatch({
                                                                        type: 'update',
                                                                        index,
                                                                        value,
                                                                    });
                                                                }
                                                            }}
                                                            validated={
                                                                isNumberInRange(timeRanges, index)
                                                                    ? ValidatedOptions.default
                                                                    : ValidatedOptions.error
                                                            }
                                                            max={maxTimeRange}
                                                            type="number"
                                                            value={timeRanges[index].value}
                                                        />
                                                    }
                                                />
                                            </div>
                                        ))}
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
            {timeRangeCounts && !isCountsDataEmpty ? (
                <AgingImagesChart
                    searchFilter={searchFilter}
                    timeRanges={timeRanges}
                    timeRangeCounts={timeRangeCounts}
                />
            ) : (
                <NoDataEmptyState />
            )}
        </WidgetCard>
    );
}

export default AgingImages;
