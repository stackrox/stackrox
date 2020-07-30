import { createContext } from 'react';
import { WorkflowState } from 'utils/WorkflowState';

const workflowStateContext = createContext(new WorkflowState());

export default workflowStateContext;
