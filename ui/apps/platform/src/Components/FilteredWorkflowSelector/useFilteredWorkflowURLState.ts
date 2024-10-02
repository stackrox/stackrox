import useURLStringUnion from 'hooks/useURLStringUnion';
import { FilteredWorkflowState, filteredWorkflowStates, SetFilteredWorkflowState } from './types';

export type FilteredWorkflowURLStateResult = {
    filteredWorkflowState: FilteredWorkflowState;
    setFilteredWorkflowState: SetFilteredWorkflowState;
};

function useFilteredWorkflowURLState(): FilteredWorkflowURLStateResult {
    const [filteredWorkflowState, setFilteredWorkflowState] = useURLStringUnion(
        'filteredWorkflowState',
        filteredWorkflowStates,
        filteredWorkflowStates[2] // @TODO: Remove this once we can show the Application and Platform views
    );

    return { filteredWorkflowState, setFilteredWorkflowState };
}

export default useFilteredWorkflowURLState;
