import React from 'react';

const defaultWorkflowStateContextData = {
    workflowStateContext: []
};

const workflowStateContext = React.createContext(defaultWorkflowStateContextData);

export default workflowStateContext;
