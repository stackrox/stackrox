import React from 'react';

const VulnMgmtPolicies = ({ selectedRowId }) => {
    return (
        <div>
            <p>{selectedRowId || 'No row selected'}</p>
        </div>
    );
};

export default VulnMgmtPolicies;
