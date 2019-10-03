import React from 'react';

const VulnMgmtNamespaces = ({ selectedRowId }) => {
    return (
        <div>
            <p>{selectedRowId || 'No row selected'}</p>
        </div>
    );
};

export default VulnMgmtNamespaces;
