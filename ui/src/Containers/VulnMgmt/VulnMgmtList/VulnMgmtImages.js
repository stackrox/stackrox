import React from 'react';

const VulnMgmtImages = ({ selectedRowId }) => {
    return (
        <div>
            <p>{selectedRowId || 'No row selected'}</p>
        </div>
    );
};

export default VulnMgmtImages;
