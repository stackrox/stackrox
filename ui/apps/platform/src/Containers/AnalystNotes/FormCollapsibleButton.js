import React from 'react';
import PropTypes from 'prop-types';
import { useQuery } from '@apollo/client';
import { TagIcon } from '@patternfly/react-icons';

import captureGraphQLErrors from 'utils/captureGraphQLErrors';
import CollapsibleCountsButton from 'Components/CollapsibleCountsButton';
import IconWithCount from 'Components/IconWithCount';
import ProcessKeyProps from './processKeyProps';
import GET_PROCESS_TAGS_COUNT from './processTagsCountQuery';

const FormCollapsibleButton = ({
    isOpen,
    onClick,
    deploymentID,
    containerName,
    execFilePath,
    args,
}) => {
    const {
        loading: isLoading,
        error,
        data = {},
    } = useQuery(GET_PROCESS_TAGS_COUNT, {
        variables: { key: { deploymentID, containerName, execFilePath, args } },
    });

    captureGraphQLErrors([error]);

    const { processTagsCount } = data;

    return (
        <CollapsibleCountsButton isOpen={isOpen} onClick={onClick}>
            <IconWithCount Icon={TagIcon} count={processTagsCount} isLoading={isLoading} />
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
