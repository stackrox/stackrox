import React from 'react';

const VulnMgmtClusters = ({ selectedRowId }) => {
    return (
        <div>
            <p>{selectedRowId || 'No row selected'}</p>
        </div>
    );
};

export default VulnMgmtClusters;
