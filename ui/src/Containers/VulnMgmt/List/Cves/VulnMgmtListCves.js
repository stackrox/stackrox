import React from 'react';

const VulnMgmtCves = ({ selectedRowId }) => {
    return (
        <div>
            <p>{selectedRowId || 'No row selected'}</p>
        </div>
    );
};

export default VulnMgmtCves;
