import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import TableCellLink from 'Components/TableCellLink';
import workflowStateContext from 'Containers/workflowStateContext';
import entityLabels from 'messages/entity';

const TableCountLink = ({ selectedRowId, entityType, textOnly, count, entityTypeText }) => {
    const workflowState = useContext(workflowStateContext);

    const type = entityTypeText || entityLabels[entityType];
    if (count === 0) return `No ${pluralize(type)}`;

    const text = `${count} ${pluralize(type, count)}`;
    if (textOnly) return text;

    const url = workflowState
        .pushListItem(selectedRowId)
        .pushList(entityType)
        .toUrl();
    return <TableCellLink pdf={textOnly} url={url} text={text} />;
};

TableCountLink.propTypes = {
    entityType: PropTypes.string.isRequired,
    selectedRowId: PropTypes.string.isRequired,
    textOnly: PropTypes.bool,
    count: PropTypes.number.isRequired,
    entityTypeText: PropTypes.string
};

TableCountLink.defaultProps = {
    textOnly: false,
    entityTypeText: null
};

export default TableCountLink;
