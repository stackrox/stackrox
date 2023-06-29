import useURLParameter, { QueryValue } from 'hooks/useURLParameter';

type PageActionResult<T extends QueryValue> = {
    pageAction: T | undefined;
    setPageAction: (action: T) => void;
};

function usePageAction<T extends QueryValue>(): PageActionResult<T> {
    const [pageActionParam, setPageActionParam] = useURLParameter('action', undefined);

    const pageAction = pageActionParam as T;

    function setPageAction(action: T): void {
        setPageActionParam(action);
    }

    return { pageAction, setPageAction };
}

export default usePageAction;
