import useURLStringUnion from 'hooks/useURLStringUnion';
import { FilteredWorkflowView, filteredWorkflowViews } from './types';

export type FilteredWorkflowViewURLStateResult = {
    filteredWorkflowView: FilteredWorkflowView;
    setFilteredWorkflowView: (value: FilteredWorkflowView) => void;
};

export const filteredWorkflowViewKey = 'filteredWorkflowView';

function useFilteredWorkflowViewURLState(
    defaultView?: FilteredWorkflowView
): FilteredWorkflowViewURLStateResult {
    const [filteredWorkflowView, setFilteredWorkflowView] = useURLStringUnion(
        filteredWorkflowViewKey,
        filteredWorkflowViews,
        defaultView
    );

    return {
        filteredWorkflowView,
        setFilteredWorkflowView,
    };
}

export default useFilteredWorkflowViewURLState;
