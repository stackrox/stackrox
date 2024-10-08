import useURLStringUnion from 'hooks/useURLStringUnion';
import { FilteredWorkflowView, filteredWorkflowViews, SetFilteredWorkflowView } from './types';

export type FilteredWorkflowViewURLStateResult = {
    filteredWorkflowView: FilteredWorkflowView;
    setFilteredWorkflowView: SetFilteredWorkflowView;
};

function useFilteredWorkflowViewURLState(): FilteredWorkflowViewURLStateResult {
    const [filteredWorkflowView, setFilteredWorkflowView] = useURLStringUnion(
        'filteredWorkflowView',
        filteredWorkflowViews
    );

    return { filteredWorkflowView, setFilteredWorkflowView };
}

export default useFilteredWorkflowViewURLState;
