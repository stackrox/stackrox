import React from 'react';
import pluralize from 'pluralize';
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
import { useQuery, useMutation } from '@apollo/client';
import Raven from 'raven-js';

import useMultiSelect from 'hooks/useMultiSelect';
import ANALYST_NOTES_TYPES from 'constants/analystnotes';
import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import getRefetchQueriesByCondition from 'Containers/AnalystNotes/analystNotesUtils/getRefetchQueriesByCondition';
import {
    getQueriesByType,
    getTagsDataByType,
} from 'Containers/AnalystNotes/AnalystTags/analystTagsQueries';

function ViolationTagsCard({
    resourceId,
    autoComplete = [],
    autoCompleteVariables,
    onInputChange,
}) {
    const variables = { resourceId };
    const { GET_TAGS, ADD_TAGS, REMOVE_TAGS } = getQueriesByType(ANALYST_NOTES_TYPES.VIOLATION);

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

    // disable input when waiting for any sort of modification
    const isDisabled = isWaitingToAddTags || isWaitingToRemoveTags || false;

    const tags = getTagsDataByType(ANALYST_NOTES_TYPES.VIOLATION, data) || [];

    const title = `${tags.length} Violation ${pluralize('Tag', tags.length)}`;

    const { isOpen: isSelectOpen, onToggle, onSelect, onClear } = useMultiSelect(onChange, tags);

    const { hasErrors, errorMessages } = captureGraphQLErrors([
        error,
        errorOnAddTags,
        errorOnRemoveTags,
    ]);

    function onChange(updatedTags) {
        const removedTags = difference(tags, updatedTags);
        const addedTags = difference(updatedTags, tags);
        if (addedTags.length) {
            addTags({
                variables: { ...variables, tags: addedTags },
            }).then(
                () => null,
                (err) => Raven.captureException(err)
            );
        }
        if (removedTags.length) {
            removeTags({
                variables: { ...variables, tags: removedTags },
            }).then(
                () => null,
                (err) => Raven.captureException(err)
            );
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
                        <Spinner />
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
}

export default ViolationTagsCard;
