import { createContext } from 'react';
import { WorkflowState } from 'modules/WorkflowState';

const workflowStateContext = createContext(new WorkflowState());

export default workflowStateContext;
