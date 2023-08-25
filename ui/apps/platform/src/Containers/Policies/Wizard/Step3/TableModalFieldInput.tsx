import React from 'react';

import ImageSigningTableModal from './ImageSigningTableModal';

function TableModalFieldInput({
    setValue,
    value,
    readOnly = false,
    tableType,
}): React.ReactElement {
    if (tableType === 'imageSigning') {
        return <ImageSigningTableModal setValue={setValue} value={value} readOnly={readOnly} />;
    }
    return <div>no such {tableType}</div>;
}

export default TableModalFieldInput;
