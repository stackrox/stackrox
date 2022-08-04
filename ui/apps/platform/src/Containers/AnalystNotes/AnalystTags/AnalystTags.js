import React from 'react';
import PropTypes from 'prop-types';
import { useQuery, useMutation } from '@apollo/client';
import difference from 'lodash/difference';
import union from 'lodash/union';
import {
    Card,
    CardHeader,
    CardBody,
    SelectOption,
    Alert,
    Select,
    SelectVariant,
    Bullseye,
    Spinner,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

import useMultiSelect from 'hooks/useMultiSelect';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import analystNotesLabels from 'messages/analystnotes';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { getQueriesByType, getTagsDataByType } from './analystTagsQueries';
import getRefetchQueriesByCondition from '../analystNotesUtils/getRefetchQueriesByCondition';
import GET_PROCESS_TAGS_COUNT from '../processTagsCountQuery';

const AnalystTags = ({ type, variables, autoComplete, autoCompleteVariables, onInputChange }) => {
    const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(type);

    const {
        loading: isLoading,
        error,
        data,
    } = useQuery(GET_TAGS, {
        variables,
    });

    // resolves once the modification + refetching happens
    const refetchAndWait = getRefetchQueriesByCondition([
        { query: GET_TAGS, variables },
        {
            query: GET_PROCESS_TAGS_COUNT,
            variables,
            exclude: type !== ANALYST_NOTES_TYPES.PROCESS,
        },
        {
            query: SEARCH_AUTOCOMPLETE_QUERY,
            variables: autoCompleteVariables,
        },
    ]);

    const [addTags, { loading: isWaitingToAddTags, error: errorOnAddTags }] = useMutation(
        ADD_TAGS,
        refetchAndWait
    );
    const [removeTags, { loading: isWaitingToRemoveTags, error: errorOnRemoveTags }] = useMutation(
        REMOVE_TAGS,
        refetchAndWait
    );

    const { hasErrors, errorMessages } = captureGraphQLErrors([
        error,
        errorOnAddTags,
        errorOnRemoveTags,
    ]);

    // disable input when waiting for any sort of modification
    const isDisabled = isWaitingToAddTags || isWaitingToRemoveTags || false;

    const tags = getTagsDataByType(type, data);

    const title = `${tags.length} ${analystNotesLabels[type]} ${pluralize('Tag', tags.length)}`;
    const { isOpen: isSelectOpen, onToggle, onSelect, onClear } = useMultiSelect(onChange, tags);

    function onChange(updatedTags) {
        const removedTags = difference(tags, updatedTags);
        const addedTags = difference(updatedTags, tags);
        if (addedTags.length) {
            addTags({
                variables: { ...variables, tags: addedTags },
            });
        }
        if (removedTags.length) {
            removeTags({
                variables: { ...variables, tags: removedTags },
            });
        }
    }

    return (
        <Card isFlat>
            <CardHeader>{title}</CardHeader>
            <CardBody>
                {hasErrors && (
                    <Alert
                        variant="warning"
                        isInline
                        title="There was an issue retrieving and/or modifying tags. Please try again later."
                    >
                        {errorMessages}
                    </Alert>
                )}
                {isLoading ? (
                    <Bullseye>
                        <Spinner isSVG />
                    </Bullseye>
                ) : (
                    <Select
                        variant={SelectVariant.typeaheadMulti}
                        onChange={onChange}
                        selections={tags}
                        placeholderText="Select or create new tags."
                        onTypeaheadInputChanged={onInputChange}
                        isCreatable
                        isDisabled={isDisabled}
                        onToggle={onToggle}
                        onSelect={onSelect}
                        onClear={onClear}
                        isOpen={isSelectOpen}
                    >
                        {union(autoComplete, tags)?.map((option) => (
                            <SelectOption key={option} value={option} />
                        ))}
                    </Select>
                )}
            </CardBody>
        </Card>
    );
};

AnalystTags.propTypes = {
    type: PropTypes.string.isRequired,
    variables: PropTypes.shape({}).isRequired,
    autoComplete: PropTypes.arrayOf(PropTypes.string),
    autoCompleteVariables: PropTypes.shape({}).isRequired,
    onInputChange: PropTypes.func.isRequired,
};

AnalystTags.defaultProps = {
    autoComplete: [],
};

export default React.memo(AnalystTags);
