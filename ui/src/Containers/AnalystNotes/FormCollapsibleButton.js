import React from 'react';
import PropTypes from 'prop-types';
import { useQuery } from 'react-apollo';
import { MessageSquare, Tag } from 'react-feather';

import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import CollapsibleCountsButton from 'Components/CollapsibleCountsButton';
import IconWithCount from 'Components/IconWithCount';
import ProcessKeyProps from './processKeyProps';
import GET_PROCESS_COMMENTS_TAGS_COUNT from './processCommentsTagsQuery';

const FormCollapsibleButton = ({
    isOpen,
    onClick,
    deploymentID,
    containerName,
    execFilePath,
    args,
}) => {
    const { loading: isLoading, error, data = {} } = useQuery(GET_PROCESS_COMMENTS_TAGS_COUNT, {
        variables: { key: { deploymentID, containerName, execFilePath, args } },
    });

    captureGraphQLErrors([error]);

    const { processCommentsCount, processTagsCount } = data;

    return (
        <CollapsibleCountsButton isOpen={isOpen} onClick={onClick}>
            <IconWithCount
                Icon={MessageSquare}
                count={processCommentsCount}
                isLoading={isLoading}
            />
            <IconWithCount Icon={Tag} count={processTagsCount} isLoading={isLoading} />
        </CollapsibleCountsButton>
    );
};

FormCollapsibleButton.propTypes = {
    isOpen: PropTypes.bool,
    onClick: PropTypes.func.isRequired,
    ...ProcessKeyProps,
};

FormCollapsibleButton.defaultProps = {
    isOpen: false,
};

export default FormCollapsibleButton;
