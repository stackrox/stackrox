import { createContext } from 'react';
import { WorkflowState } from 'modules/WorkflowStateManager';

const workflowStateContext = createContext(new WorkflowState());

export default workflowStateContext;
